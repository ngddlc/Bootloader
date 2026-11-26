package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// CalculateSHA256 вычисляет контрольную сумму файла потоково через буфер 128 КБ
func CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл для расчёта SHA-256: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	buf := make([]byte, 128*1024)

	if _, err := io.CopyBuffer(hasher, file, buf); err != nil {
		return "", fmt.Errorf("ошибка вычисления контрольной суммы: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
