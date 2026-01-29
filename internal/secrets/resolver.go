// ABOUTME: Secret resolution chain with multiple backend support
// ABOUTME: Tries resolvers in order until one succeeds
package secrets

import (
	"errors"
)

// Resolver can resolve a secret reference to its value
type Resolver interface {
	// Name returns the resolver's identifier (e.g., "env", "1password")
	Name() string

	// Available returns true if this resolver can be used
	Available() bool

	// Resolve attempts to get the secret value for the given reference
	Resolve(ref string) (string, error)
}

// Chain holds multiple resolvers and tries them in order
type Chain struct {
	resolvers []Resolver
}

// NewChain creates a new resolution chain with the given resolvers
func NewChain(resolvers ...Resolver) *Chain {
	return &Chain{resolvers: resolvers}
}

// Resolve tries each resolver in order until one succeeds
// Returns the value, the name of the resolver that succeeded, and any error
func (c *Chain) Resolve(ref string) (string, string, error) {
	if len(c.resolvers) == 0 {
		return "", "", errors.New("no resolvers configured")
	}

	var lastErr error
	for _, r := range c.resolvers {
		if !r.Available() {
			continue
		}

		value, err := r.Resolve(ref)
		if err != nil {
			lastErr = err
			continue
		}

		return value, r.Name(), nil
	}

	if lastErr != nil {
		return "", "", lastErr
	}

	return "", "", errors.New("no available resolvers could resolve the secret")
}
