package httpserver

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/bitmagnet-io/bitmagnet/internal/httpserver"
	"github.com/bitmagnet-io/bitmagnet/internal/lazy"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Schema lazy.Lazy[graphql.ExecutableSchema]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Option httpserver.Option `group:"http_server_options"`
}

func New(p Params) Result {
	return Result{
		Option: &builder{
			schema: p.Schema,
		},
	}
}

type builder struct {
	schema lazy.Lazy[graphql.ExecutableSchema]
}

func (builder) Key() string {
	return "graphql"
}

func (b builder) Apply(e *gin.Engine) error {
	schema, err := b.schema.Get()
	if err != nil {
		return err
	}

	gql := newServer(schema)

	e.POST("/graphql", func(c *gin.Context) {
		gql.ServeHTTP(c.Writer, c.Request)
	})

	pg := playground.Handler("GraphQL playground", "/graphql")

	e.GET("/graphql", func(c *gin.Context) {
		pg.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

func newServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.AroundOperations(queryTimeout(defaultQueryTimeout))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}

// defaultQueryTimeout bounds how long a GraphQL query (e.g. a torrent content search) may run.
const defaultQueryTimeout = time.Minute

// queryTimeout applies a timeout to query operations, so that a pathological search can't hold a database
// connection indefinitely. Mutations and subscriptions are not affected.
func queryTimeout(timeout time.Duration) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		if !graphql.HasOperationContext(ctx) || graphql.GetOperationContext(ctx).Operation == nil ||
			graphql.GetOperationContext(ctx).Operation.Operation != ast.Query {
			return next(ctx)
		}

		responseHandler := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			return responseHandler(ctx)
		}
	}
}
