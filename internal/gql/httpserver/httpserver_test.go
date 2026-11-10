package httpserver

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestQueryTimeout(t *testing.T) {
	t.Parallel()

	middleware := queryTimeout(time.Minute)

	run := func(op ast.Operation) (hasDeadline bool) {
		ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
			Operation: &ast.OperationDefinition{Operation: op},
		})

		responseHandler := middleware(ctx, func(context.Context) graphql.ResponseHandler {
			return func(ctx context.Context) *graphql.Response {
				_, hasDeadline = ctx.Deadline()
				return &graphql.Response{}
			}
		})

		assert.NotNil(t, responseHandler(ctx))

		return hasDeadline
	}

	assert.True(t, run(ast.Query), "queries should have a timeout")
	assert.False(t, run(ast.Mutation), "mutations should not have a timeout")
	assert.False(t, run(ast.Subscription), "subscriptions should not have a timeout")
}
