package main

import (
	"fmt"
	"github.com/lucas-clemente/quic-go/example/oauth"
)

func main() {
	oauth.DecodeRsaToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJyb290Iiwic2NvcGUiOlsicmVhZCIsIndyaXRlIiwiYWxsIiwidXNlciJdLCJtZW1iZXJOYW1lIjoicm9vdCIsImV4cCI6MTYxOTA3ODkxMCwianRpIjoiMTY5NzEwYjItMDk1MC00ZGQ5LWJlMjUtODFjNjIxM2M2ZDJjIiwiY2xpZW50X2lkIjoiY2xpZW50X3dpbmRvd3MiLCJtZW1iZXJJZCI6MjB9.WmiruMj0_FspsIqrRN8VFLaUJyAoqEWEGCUxBC2jlehhIspI3MMKXXLx17NhvdRJS5VRyVrE10sJH7a0gRyXoGZjuPfQNty4WqkxV4-fx_O3W9YQABaWvaMuPb5hCG7RcmauN-YljI777rWKg8voSHPIQ7N-KF6geM8Khk8e0upx1ru03_NALZMqowu7tgprVFZ_j0ArPmDLbofP3HEpFEOBx5PEcu0LeorHAC7IOEXyltJiShRU3OAP1EfQLEj52fNF1V3HiW0XM-CReOiyhLEcwP4VkzY3DqFyFDkLtAGG8AgaKMgsuJpcT7_5FXkL1P_7m8-dlGwFa23whcC0Gw")
	fmt.Println("test")
}