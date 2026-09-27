package service

import (
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func TestTrainingTransitions(t *testing.T) {
	if !canTransitionTraining(domain.TrainingStatusScheduled, domain.TrainingStatusCompleted) {
		t.Fatal("scheduled -> completed should be allowed")
	}
	if !canTransitionTraining(domain.TrainingStatusScheduled, domain.TrainingStatusCancelled) {
		t.Fatal("scheduled -> cancelled should be allowed")
	}
	if canTransitionTraining(domain.TrainingStatusCompleted, domain.TrainingStatusScheduled) {
		t.Fatal("completed -> scheduled must be rejected")
	}
	if canTransitionTraining(domain.TrainingStatusCancelled, domain.TrainingStatusCompleted) {
		t.Fatal("cancelled -> completed must be rejected")
	}
}

func TestTrainingTimes(t *testing.T) {
	start := time.Now()
	if err := validateTrainingTimes(start, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := validateTrainingTimes(start, start); err == nil {
		t.Fatal("same start/end must fail")
	}
	if err := validateTrainingTimes(start, start.Add(9*time.Hour)); err == nil {
		t.Fatal("overlong training must fail")
	}
}

func TestNormalizeProduct(t *testing.T) {
	duration := 30
	sessions := 8
	subType := domain.SubTypeMonthly
	input := domain.CreateProductInput{
		Name: " Месячный ", Price: 1000, Category: domain.CategorySubscription,
		SubType: &subType, DurationDays: &duration, SessionsCount: &sessions,
	}
	if err := normalizeProductInput(&input); err != nil {
		t.Fatal(err)
	}
	if input.Name != "Месячный" {
		t.Fatalf("name not trimmed: %q", input.Name)
	}

	bad := domain.CreateProductInput{Name: "Bad", Price: 10, Category: domain.CategorySubscription}
	if err := normalizeProductInput(&bad); err == nil {
		t.Fatal("subscription without subtype/duration must fail")
	}

	sports := domain.CreateProductInput{
		Name: "Мяч", Price: 100, Category: domain.CategorySports,
		SubType: &subType, DurationDays: &duration, SessionsCount: &sessions,
	}
	if err := normalizeProductInput(&sports); err != nil {
		t.Fatal(err)
	}
	if sports.SubType != nil || sports.DurationDays != nil || sports.SessionsCount != nil {
		t.Fatal("sports product must not retain subscription-only fields")
	}
}

func TestChunkMarkdownAvoidsOverlapOnlyChunk(t *testing.T) {
	content := "Первый короткий абзац.\n\n" + string(make([]byte, 0))
	content += "Очень длинный абзац " + repeatRunes("я", 80)
	chunks := chunkMarkdown(content, 40, 8)
	if len(chunks) < 3 {
		t.Fatalf("expected multiple chunks, got %d: %#v", len(chunks), chunks)
	}
	for i, chunk := range chunks {
		if chunk == "" {
			t.Fatalf("chunk %d is empty", i)
		}
		if len([]rune(chunk)) <= 8 && i > 0 {
			t.Fatalf("unexpected overlap-only chunk %d: %q", i, chunk)
		}
	}
}

func TestAISafetyIdentifierIsStableAndPrivate(t *testing.T) {
	a := aiSafetyIdentifier(42)
	b := aiSafetyIdentifier(42)
	c := aiSafetyIdentifier(43)
	if a != b || a == c {
		t.Fatalf("unexpected safety identifiers: %q %q %q", a, b, c)
	}
	if len(a) != 64 {
		t.Fatalf("unexpected safety identifier length: %d", len(a))
	}
}

func repeatRunes(value string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += value
	}
	return out
}
