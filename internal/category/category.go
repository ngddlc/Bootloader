package category

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Category описывает тематическую категорию файлов
type Category struct {
	ID         int      `json:"id"`          // Первичный ключ (1, 2, 3...)
	Name       string   `json:"name"`        // "Архивы", "Документы" и т.д.
	FolderName string   `json:"folder_name"` // Имя подпапки на диске
	Icon       string   `json:"icon"`        // Иконка/эмодзи для UI
	Extensions []string `json:"extensions"`  // Список расширений
}

// Предопределенный список категорий согласно ТЗ курсового проекта
var Categories = []Category{
	{
		ID:         1,
		Name:       "Архивы",
		FolderName: "Архивы",
		Icon:       "📦",
		Extensions: []string{".zip", ".tar.gz", ".tar", ".gz", ".rar", ".7z", ".bz2", ".xz", ".tgz", ".iso"},
	},
	{
		ID:         2,
		Name:       "Документы",
		FolderName: "Документы",
		Icon:       "📄",
		Extensions: []string{".pdf", ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".txt", ".rtf", ".odt", ".csv", ".md"},
	},
	{
		ID:         3,
		Name:       "Медиа",
		FolderName: "Медиа",
		Icon:       "🎬",
		Extensions: []string{
			".mp4", ".mkv", ".avi", ".mov", ".webm", ".flv",
			".mp3", ".wav", ".flac", ".ogg", ".aac",
			".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp",
		},
	},
	{
		ID:         4,
		Name:       "Программы",
		FolderName: "Программы",
		Icon:       "⚙️",
		Extensions: []string{".exe", ".msi", ".apk", ".dmg", ".deb", ".rpm", ".appimage"},
	},
	{
		ID:         5,
		Name:       "Другое",
		FolderName: "Другое",
		Icon:       "📁",
		Extensions: []string{},
	},
}

// Detect определяет категорию файла по его имени/расширению
func Detect(filename string) Category {
	lower := strings.ToLower(filename)

	for _, cat := range Categories {
		for _, ext := range cat.Extensions {
			if strings.HasSuffix(lower, ext) {
				return cat
			}
		}
	}

	// Если расширение не подошло ни под одно правило — категория "Другое"
	return Categories[len(Categories)-1]
}

// ResolveDestination вычисляет итоговый путь для сохранения файла:
// 1. Если customDir не пустой — сохраняет в выбранную пользователем папку.
// 2. Если customDir пустой — автоматически кладёт в подпапку категории (download/Архивы/...).
// 3. Защита от перезаписи: если файл с таким именем уже есть, добавляет (1), (2) и т.д.
func ResolveDestination(baseDir, customDir, filename string) (finalPath string, uniqueFilename string, cat Category, err error) {
	cat = Detect(filename)

	targetDir := customDir
	if targetDir == "" {
		targetDir = filepath.Join(baseDir, cat.FolderName)
	}

	// Создаём директорию, если её ещё нет
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", "", cat, fmt.Errorf("не удалось создать папку %s: %w", targetDir, err)
	}

	uniqueFilename = avoidCollision(targetDir, filename)
	finalPath = filepath.Join(targetDir, uniqueFilename)

	return finalPath, uniqueFilename, cat, nil
}

// avoidCollision проверяет наличие файла с таким именем и при конфликте возвращает "name (1).ext"
func avoidCollision(dir, filename string) string {
	targetPath := filepath.Join(dir, filename)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return filename
	}

	// Разделяем имя на основу и расширение
	ext := filepath.Ext(filename)
	stem := strings.TrimSuffix(filename, ext)

	// Особый случай для составных архивов .tar.gz
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".tar.gz") {
		ext = ".tar.gz"
		stem = filename[:len(filename)-len(".tar.gz")]
	}

	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newName
		}
	}
}
