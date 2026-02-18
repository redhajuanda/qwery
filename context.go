package qwery

type contextKey string

var (
	contextKeyTx = contextKey("tx") // contextKeyTx is a context key used to store the transaction in the context.
)
