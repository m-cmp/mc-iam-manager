package util

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/m-cmp/mc-iam-manager/constants"
)

// JSON 커스텀 타입 정의
type JSON json.RawMessage

// StringToUint는 문자열을 uint 타입으로 변환합니다.
// 변환에 실패하거나 음수일 경우 0과 에러를 반환합니다.
func StringToUint(s string) (uint, error) {
	parsedInt, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsedInt), nil
}

func UintToString(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

// check value in array 함수
func CheckValueInArray(list []string, value string) bool {
	// roleType이 reqRoleType에 포함되어 있는지 확인
	found := false
	for _, v1 := range list {
		if v1 == value {
			found = true
			break
		}
	}

	// roleType이 없으면 return
	if !found {
		fmt.Println("값이 array에 없습니다.")
		return false
	}

	// roleType이 있을 경우의 처리
	fmt.Println("값이 array에에 있습니다:")
	return true
}

// CheckValueInArrayIAMRoleType IAMRoleType 타입을 위한 배열 검사 함수
func CheckValueInArrayIAMRoleType(list []constants.IAMRoleType, value constants.IAMRoleType) bool {
	// roleType이 reqRoleType에 포함되어 있는지 확인
	found := false
	for _, v1 := range list {
		if v1 == value {
			found = true
			break
		}
	}

	// roleType이 없으면 return
	if !found {
		fmt.Println("값이 array에 없습니다.")
		return false
	}

	// roleType이 있을 경우의 처리
	fmt.Println("값이 array에에 있습니다:")
	return true
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("null point exception")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// delimiter 뒤의 문자열 추출. 없으면 원본 문자열 반환
func GetAfterDelimiter(s, delimiter string) string {
	_, after, found := strings.Cut(s, delimiter) // strings.Cut은 Go 1.18+ 에서 사용 가능합니다.
	if found {
		return after
	}
	return s // 구분자가 없으면 원본 문자열 반환
}

// LoadEnvFiles loads .env files from multiple locations for compatibility
// between local development and Docker environments
func LoadEnvFiles() {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Failed to get current working directory: %v", err)
		currentDir = "."
	}

	// Try loading from parent directory (for local development from src/)
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err == nil {
		log.Printf("✅ .env 파일을 상위 디렉토리에서 로드했습니다: %s", envPath)
		return
	} else {
		log.Printf("❌ .env 파일을 상위 디렉토리에서 로드하는데 실패했습니다: %s - %v", envPath, err)
	}

	// Try loading from current directory (for Docker compatibility)
	if err := godotenv.Load(".env"); err == nil {
		log.Printf("✅ .env 파일을 현재 디렉토리에서 로드했습니다: .env")
		return
	} else {
		log.Printf("❌ .env 파일을 현재 디렉토리에서 로드하는데 실패했습니다: .env - %v", err)
	}

	// Try loading from project root (for local development)
	// If we're in src directory, go up one level
	if strings.HasSuffix(currentDir, "src") {
		projectRoot := filepath.Join(currentDir, "..")
		rootEnvPath := filepath.Join(projectRoot, ".env")
		if err := godotenv.Load(rootEnvPath); err == nil {
			log.Printf("✅ .env 파일을 프로젝트 루트에서 로드했습니다: %s", rootEnvPath)
			return
		} else {
			log.Printf("❌ .env 파일을 프로젝트 루트에서 로드하는데 실패했습니다: %s - %v", rootEnvPath, err)
		}
	}

	// Try loading from root directory (for Docker when .env is copied to /)
	if err := godotenv.Load("/.env"); err == nil {
		log.Printf("✅ .env 파일을 루트 디렉토리에서 로드했습니다: /.env")
		return
	} else {
		log.Printf("❌ .env 파일을 루트 디렉토리에서 로드하는데 실패했습니다: /.env - %v", err)
	}

	// If we reach here, no .env file was found
	log.Printf("⚠️  .env 파일을 찾을 수 없습니다. 환경 변수를 직접 설정하거나 .env 파일을 생성해주세요.")
}

// GetAssetPath returns the appropriate asset path based on the execution environment
func GetAssetPath() string {
	// Check if we're running in Docker (current directory has asset folder)
	if _, err := os.Stat("asset"); err == nil {
		return "asset"
	}

	// Check if we're running from src directory (parent directory has asset folder)
	if _, err := os.Stat("../asset"); err == nil {
		return "../asset"
	}

	// Default fallback
	return "asset"
}
