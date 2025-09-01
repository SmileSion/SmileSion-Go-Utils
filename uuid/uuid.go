package uuid

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// UUID 类型
type UUID string

// ======================= UUID v4 (随机) =======================
func NewV4() (UUID, error) {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return "", err
	}

	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant

	return UUID(fmt.Sprintf("%08x-%04x-%04x-%04x-%04x%08x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:12], u[12:16])), nil
}

// ======================= UUID v1 (时间戳+MAC) =======================
func NewV1() (UUID, error) {
	u := make([]byte, 16)

	// 时间戳: 100-nanoseconds since 1582-10-15
	t := time.Now().UTC()
	// UUID v1 timestamp = 60 bits
	ts := uint64(t.UnixNano()/100) + 0x01B21DD213814000

	// 填充时间戳
	u[0] = byte(ts >> 24)
	u[1] = byte(ts >> 16)
	u[2] = byte(ts >> 8)
	u[3] = byte(ts)

	u[4] = byte(ts >> 40)
	u[5] = byte(ts >> 32)
	u[6] = (byte(ts>>56) & 0x0f) | 0x10 // version 1

	// 随机 node (模拟 MAC)
	node := make([]byte, 6)
	_, err := rand.Read(node)
	if err != nil {
		return "", err
	}
	u[10] = node[0]
	u[11] = node[1]
	u[12] = node[2]
	u[13] = node[3]
	u[14] = node[4]
	u[15] = node[5]

	// variant
	u[8] = (u[8] & 0x3f) | 0x80

	return UUID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])), nil
}

// ======================= UUID v5 (命名空间 + 名称) =======================
// namespace: 16-byte UUID, name: string
func NewV5(namespace UUID, name string) (UUID, error) {
	ns := strings.ReplaceAll(string(namespace), "-", "")
	nsBytes, err := hex.DecodeString(ns)
	if err != nil || len(nsBytes) != 16 {
		return "", fmt.Errorf("invalid namespace UUID")
	}

	h := md5.New()
	h.Write(nsBytes)
	h.Write([]byte(name))
	sum := h.Sum(nil)

	sum[6] = (sum[6] & 0x0f) | 0x50 // version 5
	sum[8] = (sum[8] & 0x3f) | 0x80 // variant

	return UUID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])), nil
}

// ======================= UUID 校验 =======================
func IsValidUUID(u string) bool {
	u = strings.ToLower(u)
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	return re.MatchString(u)
}

// ======================= Must UUID =======================
func MustV4() UUID {
	u, err := NewV4()
	if err != nil {
		panic(err)
	}
	return u
}

func MustV1() UUID {
	u, err := NewV1()
	if err != nil {
		panic(err)
	}
	return u
}

func MustV5(namespace UUID, name string) UUID {
	u, err := NewV5(namespace, name)
	if err != nil {
		panic(err)
	}
	return u
}

// ======================= 获取 MAC 地址（可选） =======================
func GetMAC() (net.HardwareAddr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		if len(i.HardwareAddr) == 6 {
			return i.HardwareAddr, nil
		}
	}
	return nil, fmt.Errorf("no MAC found")
}
