package thread_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/tests"
)

func createCategory(t *testing.T, ctx context.Context, cl *openapi.ClientWithResponses, adminSession openapi.RequestEditorFn) *openapi.Identifier {
	t.Helper()

	cat, err := cl.CategoryCreateWithResponse(ctx, openapi.CategoryInitialProps{
		Name:        "Category " + uuid.NewString(),
		Colour:      "#000",
		Description: "test category",
	}, adminSession)
	tests.Ok(t, err, cat)

	return &cat.JSON200.Id
}
