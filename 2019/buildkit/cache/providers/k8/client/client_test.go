package client

import (
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/stretchr/testify/require"
)

func TestOne(t *testing.T) {
	p := &model.Provider{
		Username: "user1",
		Password: "pass1",
		Endpoint: "end1",
		CaCert: `-----BEGIN CERTIFICATE-----
MIIDDDCCAfSgAwIBAgIRAOdGPwMbtAbfXZSl54yE1ykwDQYJKoZIhvcNAQELBQAw
LzEtMCsGA1UEAxMkNThmOGJhMGItODg2OC00NDMwLTk2OTUtMDQzNjI2MDY0MWVl
MB4XDTE4MDcyMDA4MDA1MFoXDTIzMDcxOTA5MDA1MFowLzEtMCsGA1UEAxMkNThm
OGJhMGItODg2OC00NDMwLTk2OTUtMDQzNjI2MDY0MWVlMIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEAn9LfAnZNTlRhhHahLQfgEInB1OD0oIRd52jG+6m1
qEgJhjfMXane9oPdYJuf2gIqWCa/zEw7e7DHJbRphLfgPy17zUV9KqSrSJjmAzjK
aU0U2cs/5Fy7g/QPKWj4SeotOL06Zh56Os/bilY78PgAX11ti8uwNKpWA9nyGm9j
0AMtlgcVYVpi1pSaKiGEZBygPkPMzSNrXHZNK3tVwrD/BXzE6u9hDRzyNO6kW2kG
FxtptcYdP80eAJWRPALRSjLKA9XwOnR1zuCEFbxTP37Mxv6YH+mjbDNIVJ0GlQ+L
OBZz4N7TXmocUIychmz2uTqqdVrJZmXHVN2kUxNiMxzlXQIDAQABoyMwITAOBgNV
HQ8BAf8EBAMCAgQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEA
cE9Hh209zoGdzU6wCwxmb2kDHgNOfvJxw8Nz2/lC6fT//cRsaz6U0525R7063h3w
lp6n7qPVTlrzLtSw26LlrCtPW0hbQCmNLUeaoSHNostg8q+xvFujSLx2isR/XRvx
Y2vAE8jj7+JfLNjpn0zGF1Tg7u8sjM1Ou3NkJRuNJVTgRyIeWxzFdcJwHhx3Tcpn
L1nWJkWe8Vqq0KSCovUnee3reIbUQf/q+7jKjU45ugHDgIycv/LK5TyaXfoxj6np
HYzaEVeZ7dHkfNSJo79bgrPGFLZU3C5KLOuEeS3CNI2feP2vA50P4Rzq3iyiEGeg
Fb0AqomP+ia4v+QbPCLK1w==
-----END CERTIFICATE-----
`,
	}
	_, err := Get(p)
	require.NoError(t, err)
}
