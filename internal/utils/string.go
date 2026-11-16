package utils

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789"

var rng = mrand.New(mrand.NewSource(time.Now().UnixNano()))

func DicPlusFileReturnPath(dic string, filename string, fileFormat string) string {
	if dic == "" {
		return filename
	}
	if filename == "" {
		return dic
	}

	filePath := ""
	if strings.HasSuffix(dic, "/") {
		filePath = dic + filename + "." + fileFormat
	} else {
		filePath = dic + "/" + filename + "." + fileFormat
	}
	return filePath
}

func RandString(size int) (string, error) {
	if size <= 0 {
		size = 6
	}

	b := make([]byte, size)
	charsetLen := byte(len(charset))

	// 生成随机字节
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// 映射到字符集
	for i := range b {
		b[i] = charset[b[i]%charsetLen]
	}

	return string(b), nil
}

func StringPtr2String(addr *string) string {
	if addr == nil {
		return ""
	}
	return *addr
}

func String2Ptr(val string) *string {
	if len(val) == 0 {
		return nil
	}
	return &val
}

func UrlParamJoin(originUrl string, param map[string]string) (string, error) {
	if originUrl == "" || param == nil || len(param) == 0 {
		return originUrl, nil
	}

	u, err := url.Parse(originUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range param {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func GenerateSixDigitCode() string {
	return strconv.Itoa(rng.Intn(900000) + 100000)
}

func UrlAddParam(link, key, value string) (string, error) {
	if link == "" {
		return "", nil
	}
	old, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	query := old.Query()
	query.Set(key, value)
	old.RawQuery = query.Encode()
	return old.String(), nil
}

func GetCharAndByte(str string, index int) (char rune, byteVal byte, err error) {
	// 将字符串转换为rune切片，以正确处理Unicode字符
	runes := []rune(str)

	// 检查索引是否有效
	if index < 0 || index >= len(runes) {
		return 0, 0, fmt.Errorf("index %d out of range (string length: %d)", index, len(runes))
	}

	// 获取指定位置的字符
	char = runes[index]

	// 获取该字符在原始字符串中的位置
	charPos := 0
	for i := 0; i < index; i++ {
		charPos += len(string(runes[i]))
	}

	// 获取该字符的字节值（第一个字节）
	if charPos < len(str) {
		byteVal = str[charPos]
	}

	return char, byteVal, nil
}
