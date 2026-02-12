package authServices

import (
	"backend-yonathan/src/pkg/apiresponse"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

type base64Request struct {
	Value string `json:"value"`
}

type certRequest struct {
	CommonName   string `json:"commonName"`
	Organization string `json:"organization"`
	ValidDays    int    `json:"validDays"`
	Password     string `json:"password"`
}

func EncodeBase64(c *fiber.Ctx) error {
	var payload base64Request
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{
		"input":   payload.Value,
		"encoded": base64.StdEncoding.EncodeToString([]byte(payload.Value)),
	})
}

func DecodeBase64(c *fiber.Ctx) error {
	var payload base64Request
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	decoded, err := base64.StdEncoding.DecodeString(payload.Value)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_base64", "El valor no es base64 valido", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{
		"input":   payload.Value,
		"decoded": string(decoded),
	})
}

func GenerateUUIDv4(c *fiber.Ctx) error {
	return apiresponse.Success(c, fiber.Map{
		"uuid": uuid.NewString(),
	})
}

func GenerateSelfSignedCert(c *fiber.Ctx) error {
	var payload certRequest
	if err := c.BodyParser(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	if payload.CommonName == "" {
		payload.CommonName = "localhost"
	}
	if payload.Organization == "" {
		payload.Organization = "PortfolioTools"
	}
	if payload.ValidDays <= 0 {
		payload.ValidDays = 365
	}
	if payload.Password == "" {
		payload.Password = "changeit"
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "key_generation_failed", "No se pudo generar la llave privada", err.Error())
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "serial_generation_failed", "No se pudo generar serial del certificado", err.Error())
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   payload.CommonName,
			Organization: []string{payload.Organization},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(payload.ValidDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "cert_generation_failed", "No se pudo generar el certificado", err.Error())
	}

	keyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "cert_parse_failed", "No se pudo procesar el certificado generado", err.Error())
	}

	pfxData, err := pkcs12.Encode(rand.Reader, privateKey, cert, nil, payload.Password)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "pfx_generation_failed", "No se pudo generar el archivo pfx", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{
		"certPem":    string(certPEM),
		"keyPem":     string(keyPEM),
		"certBase64": base64.StdEncoding.EncodeToString(certPEM),
		"pfxBase64":  base64.StdEncoding.EncodeToString(pfxData),
		"password":   payload.Password,
	})
}
