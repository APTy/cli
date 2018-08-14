package jose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	realx509 "crypto/x509"

	"github.com/pkg/errors"
	"github.com/smallstep/cli/crypto/pemutil"
	"github.com/smallstep/cli/crypto/randutil"
	"github.com/smallstep/cli/pkg/x509"
	"golang.org/x/crypto/ed25519"
)

// GenerateJWK generates a JWK given the key type, curve, alg, use, kid and
// the size of the RSA or oct keys if necessary.
func GenerateJWK(kty, crv, alg, use, kid string, size int) (jwk *JSONWebKey, err error) {
	switch kty {
	case "EC":
		return generateECKey(crv, alg, use, kid)
	case "RSA":
		return generateRSAKey(size, alg, use, kid)
	case "OKP":
		return generateOKPKey(crv, alg, use, kid)
	case "oct":
		return generateOctKey(size, alg, use, kid)
	default:
		return nil, errors.Errorf("missing or invalid value for flag '--kty'")
	}
}

// GenerateJWKFromPEM returns an incomplete JSONWebKey using the key from a
// PEM file.
func GenerateJWKFromPEM(filename string) (*JSONWebKey, error) {
	key, err := pemutil.Read(filename)
	if err != nil {
		return nil, err
	}

	switch key := key.(type) {
	case *rsa.PrivateKey, *rsa.PublicKey, *ecdsa.PrivateKey, *ecdsa.PublicKey, ed25519.PrivateKey, ed25519.PublicKey:
		return &JSONWebKey{
			Key:       key,
			Algorithm: algForKey(key),
		}, nil
	case *x509.Certificate:
		// have: step-cli x509 Certificate
		// want: crypto/x509 Certificate
		crt, err := realx509.ParseCertificate(key.Raw)
		if err != nil {
			return nil, err
		}
		return &JSONWebKey{
			Key:          key.PublicKey,
			Certificates: []*realx509.Certificate{crt},
			Algorithm:    algForKey(key.PublicKey),
		}, nil
	case []*x509.Certificate:
		chain := key
		if len(chain) == 0 {
			return nil, errors.New("jose/generate: no certificates in chain")
		}
		// have: step-cli x509 Certificate
		// want: crypto/x509 Certificate
		certs := make([]*realx509.Certificate, len(chain))
		for i := range chain {
			crt, err := realx509.ParseCertificate(chain[i].Raw)
			if err != nil {
				return nil, err
			}
			certs[i] = crt
		}
		leafPubkey := chain[0].PublicKey
		return &JSONWebKey{
			Key:          leafPubkey,
			Certificates: certs,
			Algorithm:    algForKey(leafPubkey),
		}, nil
	default:
		return nil, errors.Errorf("error parsing %s: unsupported key type '%T'", filename, key)
	}
}

func algForKey(key crypto.PublicKey) string {
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		return getECAlgorithm(key.Curve)
	case *ecdsa.PublicKey:
		return getECAlgorithm(key.Curve)
	case ed25519.PrivateKey, ed25519.PublicKey:
		return EdDSA
	default:
		return ""
	}
}

func generateECKey(crv, alg, use, kid string) (*JSONWebKey, error) {
	var c elliptic.Curve
	var sigAlg string
	switch crv {
	case P256, "": // default
		c, sigAlg = elliptic.P256(), ES256
	case P384:
		c, sigAlg = elliptic.P384(), ES384
	case P521:
		c, sigAlg = elliptic.P521(), ES512
	default:
		return nil, errors.Errorf("missing or invalid value for flag '--crv'")
	}

	key, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "error generating ECDSA key")
	}

	switch use {
	case "enc":
		if alg == "" {
			alg = string(DefaultECKeyAlgorithm)
		}
	default:
		if alg == "" {
			alg = sigAlg
		}
	}

	return &JSONWebKey{
		Key:       key,
		Algorithm: alg,
		Use:       use,
		KeyID:     kid,
	}, nil
}

func generateRSAKey(bits int, alg, use, kid string) (*JSONWebKey, error) {
	if bits == 0 {
		bits = DefaultRSASize
	}

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, errors.Wrap(err, "error generating RSA key")
	}

	switch use {
	case "enc":
		if alg == "" {
			alg = string(DefaultRSAKeyAlgorithm)
		}
	default:
		if alg == "" {
			alg = DefaultRSASigAlgorithm
		}
	}

	return &JSONWebKey{
		Key:       key,
		Algorithm: alg,
		Use:       use,
		KeyID:     kid,
	}, nil
}

func generateOKPKey(crv, alg, use, kid string) (*JSONWebKey, error) {
	switch crv {
	case Ed25519, "": // default
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, errors.Wrap(err, "error generating Ed25519 key")
		}

		switch use {
		case "enc":
			return nil, errors.New("invalid algorithm: Ed25519 cannot be used for encryption")
		default:
			if alg == "" {
				alg = EdDSA
			}
		}

		return &JSONWebKey{
			Key:       key,
			Algorithm: alg,
			Use:       use,
			KeyID:     kid,
		}, nil
	default:
		return nil, errors.Errorf("missing or invalid value for flag '--crv'")
	}
}

func generateOctKey(size int, alg, use, kid string) (*JSONWebKey, error) {
	if size == 0 {
		size = DefaultOctSize
	}

	key, err := randutil.Alphanumeric(size)
	if err != nil {
		return nil, err
	}

	switch use {
	case "enc":
		if alg == "" {
			alg = string(DefaultOctKeyAlgorithm)
		}
	default:
		if alg == "" {
			alg = string(DefaultOctSigsAlgorithm)
		}
	}

	return &JSONWebKey{
		Key:       []byte(key),
		Algorithm: alg,
		Use:       use,
		KeyID:     kid,
	}, nil
}
