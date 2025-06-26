package identity_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hossein1376/kamune/internal/identity"
)

func TestEd25519_SignVerify(t *testing.T) {
	a := require.New(t)
	msg := []byte("Make the world a better place")

	e, err := identity.NewEd25519()
	a.NoError(err)
	a.NotNil(e)
	pub := e.PublicKey
	a.NotNil(pub)
	sig, err := e.Sign(msg)
	a.NoError(err)
	a.NotNil(sig)

	t.Run("valid signature", func(t *testing.T) {
		verified := identity.VerifyEd25519(pub, msg, sig)
		a.True(verified)
	})
	t.Run("invalid signature", func(t *testing.T) {
		sig := slices.Clone(sig)
		sig[0] ^= 0xFF

		verified := identity.VerifyEd25519(pub, msg, sig)
		a.False(verified)
	})
	t.Run("invalid hash", func(t *testing.T) {
		msg = append(msg, []byte("!")...)

		verified := identity.VerifyEd25519(pub, msg, sig)
		a.False(verified)
	})
	t.Run("invalid public key", func(t *testing.T) {
		another, err := identity.NewEd25519()
		a.NoError(err)
		verified := identity.VerifyEd25519(another.PublicKey, msg, sig)
		a.False(verified)
	})
}
