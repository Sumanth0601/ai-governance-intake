package proposals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sumanth/ai-governance-intake/internal/config"
	"github.com/sumanth/ai-governance-intake/internal/embeddings"
	"github.com/sumanth/ai-governance-intake/internal/llm"
)

var ErrLLMParseFailure = errors.New("LLM returned unparseable JSON")

const systemPrompt = `You are an AI governance compliance analyst specializing in ISO 42001 (AI Management Systems) and NIST AI RMF.

Analyze the submitted AI project proposal and return ONLY a valid JSON object. No explanation. No markdown. No code fences.

The JSON must have this exact shape:
{
  "overall_risk": "low" | "medium" | "high" | "critical",
  "findings": [
    {
      "control": "ISO 42001 clause reference (e.g. '6.1.2 AI risk assessment')",
      "severity": "low" | "medium" | "high" | "critical",
      "issue": "one sentence describing the gap or risk",
      "recommendation": "one sentence actionable fix"
    }
  ],
  "summary": "2-3 sentence plain-English summary of the proposal's governance posture"
}

Evaluate against these controls at minimum:
- Data privacy and PII handling (ISO 42001 § 8.4)
- Human oversight and intervention capability (ISO 42001 § 8.5)
- Transparency and explainability (ISO 42001 § 6.1.2)
- Bias and fairness risk across affected populations (NIST AI RMF GOVERN 1.1)
- Intended use boundary definition (ISO 42001 § 4.1)
- High-risk AI classification (EU AI Act Annex III alignment)

Be specific — reference the actual content of the proposal, not generic advice.`

type SubmitRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DataSources string `json:"data_sources"`
	IntendedUse string `json:"intended_use"`
}

type Finding struct {
	Control        string `json:"control"`
	Severity       string `json:"severity"`
	Issue          string `json:"issue"`
	Recommendation string `json:"recommendation"`
}

type Scorecard struct {
	OverallRisk string    `json:"overall_risk"`
	Findings    []Finding `json:"findings"`
	Summary     string    `json:"summary"`
}

type Proposal struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DataSources string     `json:"data_sources"`
	IntendedUse string     `json:"intended_use"`
	RiskScore   string     `json:"risk_score"`
	Scorecard   *Scorecard `json:"scorecard,omitempty"`
	DuplicateOf *string    `json:"duplicate_of"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SubmitResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Title       string     `json:"title"`
	RiskScore   string     `json:"risk_score,omitempty"`
	Scorecard   *Scorecard `json:"scorecard,omitempty"`
	DuplicateOf *string    `json:"duplicate_of,omitempty"`
	Similarity  *float64   `json:"similarity,omitempty"`
	Message     string     `json:"message,omitempty"`
}

type Service struct {
	repo             *Repository
	embeddingsClient *embeddings.Client
	llm              *llm.Client
	cfg              *config.Config
}

func NewService(repo *Repository, emb *embeddings.Client, llmClient *llm.Client, cfg *config.Config) *Service {
	return &Service{
		repo:             repo,
		embeddingsClient: emb,
		llm:              llmClient,
		cfg:              cfg,
	}
}

func (s *Service) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	fullText := fmt.Sprintf("Title: %s\nDescription: %s\nData Sources: %s\nIntended Use: %s",
		req.Title, req.Description, req.DataSources, req.IntendedUse)

	embedding, err := s.embeddingsClient.Embed(ctx, fullText)
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}

	dupe, err := s.repo.FindSimilar(ctx, embedding, s.cfg.SimilarityThreshold)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}

	if dupe != nil {
		id, err := s.repo.InsertDuplicate(ctx, req, embedding, dupe.ID)
		if err != nil {
			return nil, fmt.Errorf("insert duplicate: %w", err)
		}
		sim := dupe.Similarity
		return &SubmitResponse{
			ID:          id,
			Status:      "duplicate",
			Title:       req.Title,
			DuplicateOf: &dupe.ID,
			Similarity:  &sim,
			Message:     "A semantically similar proposal already exists. Review the existing initiative before proceeding.",
		}, nil
	}

	rawJSON, err := s.llm.Complete(ctx, systemPrompt, fullText)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	// Strip markdown fences if the model ignores instructions
	rawJSON = stripFences(rawJSON)

	var scorecard Scorecard
	if err := json.Unmarshal([]byte(rawJSON), &scorecard); err != nil {
		return nil, ErrLLMParseFailure
	}

	proposal, err := s.repo.Insert(ctx, req, embedding, scorecard)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}

	return &SubmitResponse{
		ID:        proposal.ID,
		Status:    "scored",
		Title:     proposal.Title,
		RiskScore: proposal.RiskScore,
		Scorecard: proposal.Scorecard,
	}, nil
}

func (s *Service) List(ctx context.Context) ([]Proposal, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Proposal, error) {
	return s.repo.GetByID(ctx, id)
}

// stripFences removes ```json ... ``` wrappers that some models add despite instructions.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove first line (```json or ```)
		idx := strings.Index(s, "\n")
		if idx != -1 {
			s = s[idx+1:]
		}
		// Remove trailing ```
		if end := strings.LastIndex(s, "```"); end != -1 {
			s = s[:end]
		}
	}
	return strings.TrimSpace(s)
}
