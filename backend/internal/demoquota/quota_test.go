package demoquota

import "testing"

func TestEvaluate_SessionUnderLimit_Allowed(t *testing.T) {
	// Arrange
	sessionCount := int32(5)
	globalCount := int64(10)

	// Act
	result := evaluate(sessionCount, globalCount)

	// Assert
	if !result.Allowed {
		t.Fatalf("expected allowed, got denied with message %q", result.Message)
	}
}

func TestEvaluate_SessionOverLimit_DeniedWithSessionMessage(t *testing.T) {
	// Arrange
	sessionCount := int32(6)
	globalCount := int64(10)

	// Act
	result := evaluate(sessionCount, globalCount)

	// Assert
	if result.Allowed {
		t.Fatal("expected denied, got allowed")
	}
	if result.Message != SessionLimitMessage {
		t.Errorf("expected session limit message, got %q", result.Message)
	}
}

func TestEvaluate_GlobalOverLimit_DeniedWithGlobalMessage(t *testing.T) {
	// Arrange
	sessionCount := int32(1)
	globalCount := int64(51)

	// Act
	result := evaluate(sessionCount, globalCount)

	// Assert
	if result.Allowed {
		t.Fatal("expected denied, got allowed")
	}
	if result.Message != GlobalLimitMessage {
		t.Errorf("expected global limit message, got %q", result.Message)
	}
}

func TestEvaluate_BothOverLimit_SessionMessageTakesPrecedence(t *testing.T) {
	// Arrange
	sessionCount := int32(6)
	globalCount := int64(51)

	// Act
	result := evaluate(sessionCount, globalCount)

	// Assert
	if result.Message != SessionLimitMessage {
		t.Errorf("expected session limit message to take precedence, got %q", result.Message)
	}
}

func TestEvaluate_ExactlyAtLimits_Allowed(t *testing.T) {
	// Arrange
	sessionCount := int32(PerSessionLimit)
	globalCount := int64(GlobalLimit)

	// Act
	result := evaluate(sessionCount, globalCount)

	// Assert
	if !result.Allowed {
		t.Fatalf("expected allowed at exactly the limit, got denied with message %q", result.Message)
	}
}
