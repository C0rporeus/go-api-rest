package mailaudit

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	dkimWeakBits    = 1024 // < 1024: trivially breakable
	dkimMinGoodBits = 2048
)

// DKIMKey is a parsed DKIM key record.
type DKIMKey struct {
	Raw     string
	Type    string // rsa (default), ed25519
	Bits    int    // key size in bits (0 if unknown/revoked)
	Revoked bool   // empty p= revokes the selector
	Testing bool   // t=y
}

// DKIMProbe is the DNS outcome for a selector, evaluated by AuditDKIM.
type DKIMProbe struct {
	Selector string
	TXTs     []string
	Err      error
}

// DKIMAudit is the outcome of auditing the probed selectors.
type DKIMAudit struct {
	Found    []string // selectors that publish a key
	Keys     map[string]DKIMKey
	Findings []Finding
}

// ParseDKIM parses a DKIM key record (RFC 6376 §3.6.1). It never panics on
// arbitrary input (covered by a fuzz test).
func ParseDKIM(raw string) (DKIMKey, []Finding) {
	key := DKIMKey{Raw: raw, Type: "rsa"}
	tags := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		if name, value, ok := strings.Cut(strings.TrimSpace(part), "="); ok {
			tags[strings.ToLower(strings.TrimSpace(name))] = strings.Join(strings.Fields(value), "")
		}
	}
	if version, ok := tags["v"]; ok && !strings.EqualFold(version, "DKIM1") {
		return key, []Finding{dkimSyntax(fmt.Sprintf("v=%q no es DKIM1", truncate(version)))}
	}
	if kind, ok := tags["k"]; ok && kind != "" {
		key.Type = strings.ToLower(kind)
	}
	key.Testing = strings.Contains(strings.ToLower(tags["t"]), "y")
	publicKey, hasKey := tags["p"]
	if !hasKey {
		return key, []Finding{dkimSyntax("falta la etiqueta p= con la clave pública")}
	}
	if publicKey == "" {
		key.Revoked = true
		return key, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return key, []Finding{dkimSyntax("p= no es base64 válido")}
	}
	bits, message := keyBits(key.Type, decoded)
	key.Bits = bits
	if message != "" {
		return key, []Finding{dkimSyntax(message)}
	}
	return key, nil
}

func keyBits(kind string, decoded []byte) (int, string) {
	switch kind {
	case "ed25519":
		if len(decoded) != ed25519.PublicKeySize {
			return 0, "la clave ed25519 debe tener 32 bytes"
		}
		return 256, ""
	case "rsa":
		parsed, err := x509.ParsePKIXPublicKey(decoded)
		if err != nil {
			return 0, "la clave pública RSA no es un SubjectPublicKeyInfo válido"
		}
		switch public := parsed.(type) {
		case *rsa.PublicKey:
			return public.N.BitLen(), ""
		case *ecdsa.PublicKey:
			return 0, "la clave no es RSA aunque k=rsa"
		}
		return 0, "tipo de clave no soportado"
	}
	return 0, fmt.Sprintf("k=%q no es rsa ni ed25519", truncate(kind))
}

func dkimSyntax(detail string) Finding {
	return Finding{
		ID: "dkim.syntax", Section: SectionDKIM, Severity: SeverityCritical,
		Title:       "Registro DKIM inválido",
		Detail:      truncate(detail),
		Remediation: "Regenera el registro desde tu proveedor de correo y publícalo completo (cuidado con el límite de 255 caracteres por cadena TXT).",
	}
}
