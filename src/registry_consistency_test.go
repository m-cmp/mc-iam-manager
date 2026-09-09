package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// mcmp API 레지스트리 정합성 검증
//
// 전달 체인은 다음과 같다:
//
//	핸들러 @Router 주석 -> swagger.json -> service-actions.yaml(시드) -> mcmp_api_actions(DB) -> mc-web-console BFF
//
// BFF는 레지스트리 resourcePath로 백엔드 URL을 조립하므로, 이 체인 어디서든 값이 어긋나면
// 콘솔 기능이 그대로 404가 된다(WEB-BUG-052). 게다가 swag은 라우트 등록 여부를 검증하지 않아
// 주석만 틀려도 유령 경로가 레지스트리까지 전파된다.
//
// 이 테스트는 main.go의 실제 라우트를 진실로 삼아 3단(라우트 <-> swagger <-> 시드)을 대조한다.
// 경로뿐 아니라 메서드까지 비교한다 — 경로만 보면 "경로는 있는데 그 메서드가 없어" 405가 나는
// 케이스(deleteCspRole)를 놓친다.

type endpoint struct {
	method string
	path   string
}

func (e endpoint) String() string { return strings.ToUpper(e.method) + " " + e.path }

var (
	reGroup   = regexp.MustCompile(`(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"`)
	reRoute   = regexp.MustCompile(`\b(\w+)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)"\s*,\s*(\w+)\.(\w+)`)
	reBase    = regexp.MustCompile(`basePath\s*:?=\s*"([^"]*)"`)
	rePathVar = regexp.MustCompile(`:([A-Za-z0-9_]+)`)
)

// toSwaggerPath echo 표기(:param)를 swagger 표기({param})로 변환한다.
func toSwaggerPath(p string) string {
	p = rePathVar.ReplaceAllString(p, "{$1}")
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// parseRoutes main.go에 등록된 라우트를 핸들러 함수명 기준으로 수집한다.
// 라우트 등록이 전부 main() 안에 있어 함수로 재사용할 수 없으므로 소스를 파싱한다.
func parseRoutes(t *testing.T) map[string][]endpoint {
	t.Helper()
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("main.go 읽기 실패: %v", err)
	}
	code := string(src)

	base := "/api"
	if m := reBase.FindStringSubmatch(code); m != nil {
		base = m[1]
	}
	groups := map[string]string{"api": base}
	for _, m := range reGroup.FindAllStringSubmatch(code, -1) {
		v, parent, path := m[1], m[2], m[3]
		if parent == "e" {
			continue // 루트 그룹은 prefix 없음
		}
		groups[v] = groups[parent] + path
	}

	routes := map[string][]endpoint{}
	for _, m := range reRoute.FindAllStringSubmatch(code, -1) {
		v, method, path, fn := m[1], strings.ToLower(m[2]), m[3], m[5]
		prefix, ok := groups[v]
		if !ok {
			if v != "e" {
				continue // 라우터 변수가 아님
			}
			prefix = "" // e.GET("/readyz", ...) 처럼 루트 등록
		}
		routes[fn] = append(routes[fn], endpoint{method, toSwaggerPath(prefix + path)})
	}
	if len(routes) == 0 {
		t.Fatal("main.go에서 라우트를 하나도 파싱하지 못했다 — 파서가 깨졌을 수 있다")
	}
	return routes
}

// parseAnnotations 핸들러의 @Router 주석을 (operationId, endpoint)로 수집한다.
func parseAnnotations(t *testing.T) (map[string]endpoint, map[string]string) {
	t.Helper()
	entries, err := os.ReadDir("handler")
	if err != nil {
		t.Fatalf("handler 디렉토리 읽기 실패: %v", err)
	}

	reRouter := regexp.MustCompile(`^\s*//\s*@Router\s+(\S+)\s+\[(\w+)\](.*)$`)
	reID := regexp.MustCompile(`^\s*//\s*@Id\s+(\S+)`)
	reFunc := regexp.MustCompile(`^func\s+\([^)]*\)\s+(\w+)\(`)

	anns := map[string]endpoint{}  // operationId -> 주석 경로
	owner := map[string]string{}   // operationId -> 핸들러 함수명
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile("handler/" + name)
		if err != nil {
			t.Fatalf("%s 읽기 실패: %v", name, err)
		}
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			m := reRouter.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			path, method, trailing := m[1], strings.ToLower(m[2]), strings.TrimSpace(m[3])
			loc := fmt.Sprintf("%s:%d", name, i+1)

			if trailing != "" {
				t.Errorf("%s: @Router 뒤에 인라인 주석이 붙어 있다 (%q) — swag 파싱을 깨뜨릴 수 있다", loc, trailing)
			}
			if strings.Contains(path, ":") {
				t.Errorf("%s: @Router에 echo 표기 %q가 쓰였다 — swagger 표기 {param}을 써야 시드에 리터럴 ':param'이 박히지 않는다", loc, path)
				path = toSwaggerPath(path)
			}

			// 같은 주석 블록에서 @Id와 함수명을 찾는다.
			opID, fn := "", ""
			for j := i; j < len(lines) && j < i+20; j++ {
				if fn == "" {
					if fm := reFunc.FindStringSubmatch(lines[j]); fm != nil {
						fn = fm[1]
						break
					}
				}
				if im := reID.FindStringSubmatch(lines[j]); im != nil && opID == "" {
					opID = im[1]
				}
			}
			if opID == "" {
				t.Errorf("%s: @Router가 있으나 @Id가 없다 — operationId 없이는 레지스트리에 등록되지 않는다", loc)
				continue
			}
			anns[opID] = endpoint{method, path}
			owner[opID] = fn
		}
	}
	return anns, owner
}

// TestRegistryConsistency 라우트 <-> swagger <-> 시드 3단 대조.
func TestRegistryConsistency(t *testing.T) {
	routes := parseRoutes(t)
	anns, owner := parseAnnotations(t)

	// ── 1단: @Router 주석 <-> main.go 실제 라우트 ──
	for opID, ann := range anns {
		fn := owner[opID]
		regs := routes[fn]
		if len(regs) == 0 {
			t.Errorf("%s(%s): @Router는 %s를 가리키는데 main.go에 등록된 라우트가 없다 — 유령 경로가 레지스트리로 전파된다. 라우트를 등록하거나 @Router/@Id를 제거하라",
				opID, fn, ann)
			continue
		}
		matched := false
		for _, r := range regs {
			if r == ann {
				matched = true
				break
			}
		}
		if !matched {
			got := make([]string, 0, len(regs))
			for _, r := range regs {
				got = append(got, r.String())
			}
			t.Errorf("%s(%s): @Router=%s 이지만 실제 등록은 %s — 실제 라우트가 진실이다(핸들러의 c.Param 이름으로 확인할 것)",
				opID, fn, ann, strings.Join(got, ", "))
		}
	}

	// ── 2단: swagger.json <-> @Router 주석 ──
	swagger := loadSwagger(t)
	for opID, sw := range swagger {
		ann, ok := anns[opID]
		if !ok {
			t.Errorf("%s: swagger에는 있으나 핸들러 주석에서 찾지 못했다 — swag 재생성이 필요할 수 있다", opID)
			continue
		}
		if sw != ann {
			t.Errorf("%s: swagger=%s, 주석=%s — `cd src && swag init --output ./docs` 재생성 후 docs/에도 복사하라", opID, sw, ann)
		}
	}
	for opID := range anns {
		if _, ok := swagger[opID]; !ok {
			t.Errorf("%s: 주석에는 있으나 swagger에 없다 — swag 재생성이 필요하다", opID)
		}
	}

	// ── 3단: service-actions.yaml(시드) <-> swagger ──
	seedPaths := []string{"../asset/mcmpapi/service-actions.yaml", "../conf/mc-iam-manager/service-actions.yaml"}
	for _, p := range seedPaths {
		seed := loadSeed(t, p)
		for opID, sw := range swagger {
			sd, ok := seed[opID]
			if !ok {
				t.Errorf("%s: %s에 없다 — tool/swagger-to-actions로 시드를 재생성하라", opID, p)
				continue
			}
			if sd != sw {
				t.Errorf("%s: 시드=%s, swagger=%s (%s) — 시드 재생성 누락", opID, sd, sw, p)
			}
		}
		extra := make([]string, 0)
		for opID := range seed {
			if _, ok := swagger[opID]; !ok {
				extra = append(extra, opID)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("%s: swagger에 없는 액션이 남아 있다 %v — 시드를 재생성하라(--append는 삭제분을 지우지 못한다)", p, extra)
		}
	}
}

func loadSwagger(t *testing.T) map[string]endpoint {
	t.Helper()
	raw, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		t.Fatalf("docs/swagger.json 읽기 실패: %v", err)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("swagger.json 파싱 실패: %v", err)
	}
	out := map[string]endpoint{}
	for path, ops := range doc.Paths {
		for method, op := range ops {
			if op.OperationID != "" {
				out[op.OperationID] = endpoint{strings.ToLower(method), path}
			}
		}
	}
	return out
}

func loadSeed(t *testing.T, path string) map[string]endpoint {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s 읽기 실패: %v", path, err)
	}
	var doc struct {
		ServiceActions map[string]map[string]struct {
			Method       string `yaml:"method"`
			ResourcePath string `yaml:"resourcePath"`
		} `yaml:"serviceActions"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s 파싱 실패: %v", path, err)
	}
	out := map[string]endpoint{}
	for opID, spec := range doc.ServiceActions["mc-iam-manager"] {
		if opID == "_meta" {
			continue
		}
		out[opID] = endpoint{strings.ToLower(spec.Method), spec.ResourcePath}
	}
	return out
}
