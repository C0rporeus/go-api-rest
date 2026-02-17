package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
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

// EncodeBase64 godoc
// @Summary      Codificar Base64
// @Description  Codifica un string en Base64
// @Tags         Tools
// @Accept       json
// @Produce      json
// @Param        payload  body  object{value=string}  true  "Texto a codificar"
// @Success      200  {object}  map[string]string  "input, encoded"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/base64/encode [post]
func EncodeBase64(c fiber.Ctx) error {
	var payload base64Request
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	if len(payload.Value) > constants.MaxBase64InputSize {
		return apiresponse.Error(c, fiber.StatusBadRequest, "input_too_large",
			"El texto excede el tamano maximo permitido", nil)
	}

	return apiresponse.Success(c, fiber.Map{
		"input":   payload.Value,
		"encoded": base64.StdEncoding.EncodeToString([]byte(payload.Value)),
	})
}

// DecodeBase64 godoc
// @Summary      Decodificar Base64
// @Description  Decodifica un string Base64
// @Tags         Tools
// @Accept       json
// @Produce      json
// @Param        payload  body  object{value=string}  true  "Texto en Base64"
// @Success      200  {object}  map[string]string  "input, decoded"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/base64/decode [post]
func DecodeBase64(c fiber.Ctx) error {
	var payload base64Request
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	if len(payload.Value) > constants.MaxBase64InputSize {
		return apiresponse.Error(c, fiber.StatusBadRequest, "input_too_large",
			"El texto excede el tamano maximo permitido", nil)
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

// GenerateUUIDv4 godoc
// @Summary      Generar UUID v4
// @Description  Genera un UUID v4 aleatorio
// @Tags         Tools
// @Produce      json
// @Success      200  {object}  map[string]string  "uuid"
// @Router       /api/tools/uuid/v4 [get]
func GenerateUUIDv4(c fiber.Ctx) error {
	return apiresponse.Success(c, fiber.Map{
		"uuid": uuid.NewString(),
	})
}

// GenerateSelfSignedCert godoc
// @Summary      Generar certificado autofirmado
// @Description  Genera un certificado X.509 autofirmado con PEM, DER y PFX
// @Tags         Tools
// @Accept       json
// @Produce      json
// @Param        payload  body  object{commonName=string,organization=string,validDays=int,password=string}  true  "Parametros del certificado"
// @Success      200  {object}  map[string]string  "certPem, keyPem, certBase64, pfxBase64, password"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/tools/certs/self-signed [post]
func GenerateSelfSignedCert(c fiber.Ctx) error {
	var payload certRequest
	if err := c.Bind().Body(&payload); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	payload.CommonName = sanitizer.SanitizePlainText(strings.TrimSpace(payload.CommonName), constants.MaxCertCommonName)
	payload.Organization = sanitizer.SanitizePlainText(strings.TrimSpace(payload.Organization), constants.MaxCertOrgLength)

	if payload.CommonName == "" {
		payload.CommonName = constants.DefaultCertCommonName
	}
	if payload.Organization == "" {
		payload.Organization = constants.DefaultCertOrganization
	}
	if payload.ValidDays <= 0 {
		payload.ValidDays = constants.DefaultCertValidDays
	}
	if payload.ValidDays > constants.MaxCertValidDays {
		payload.ValidDays = constants.MaxCertValidDays
	}
	if payload.Password == "" {
		payload.Password = constants.DefaultCertPassword
	}
	if len(payload.Password) > constants.MaxPasswordLength {
		return apiresponse.Error(c, fiber.StatusBadRequest, "password_too_long",
			"La contrasena del certificado excede el largo maximo", nil)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, constants.DefaultCertKeyBits)
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
