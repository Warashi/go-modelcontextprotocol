package router

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ----------------------------------------
// エラー定義
// ----------------------------------------
var ErrNotFound = errors.New("route not found")

// ----------------------------------------
// Request 型
// ----------------------------------------
type Request struct {
	Query  map[string]string
	Params map[string]string
}

// ----------------------------------------
// Handler[T] インターフェース
// ----------------------------------------
type Handler[T any] interface {
	Handle(ctx context.Context, req *Request) (T, error)
}

// HandlerFunc[T] は関数を Handler[T] として扱うための型
type HandlerFunc[T any] func(ctx context.Context, req *Request) (T, error)

// インターフェース実装
func (f HandlerFunc[T]) Handle(ctx context.Context, req *Request) (T, error) {
	return f(ctx, req)
}

// ----------------------------------------
// 内部構造：登録ルートを表す struct
// ----------------------------------------
type route[T any] struct {
	scheme string // 小文字で保持（固定値のみ）
	// host が動的かどうか
	hostIsParam   bool
	hostParamName string // hostIsParam == true の場合に使用
	host          string // hostIsParam == false の場合に使用（小文字化）
	// パス情報
	pathSegments []pathSegment
	// クエリ情報（固定キー・固定値のみ）
	query map[string]string
	// ハンドラ
	handler Handler[T]
}

// pathSegment は静的 or 動的セグメントを区別する
type pathSegment struct {
	isParam   bool
	paramName string // isParam = true の場合に使用
	literal   string // isParam = false の場合に使用
}

// ----------------------------------------
// Mux[T] 本体
// ----------------------------------------
type Mux[T any] struct {
	routes          []route[T]
	notFoundHandler Handler[T]
}

// ----------------------------------------
// Mux[T] のコンストラクタ（必要に応じて）
// ----------------------------------------
func NewMux[T any]() *Mux[T] {
	return &Mux[T]{}
}

// ----------------------------------------
// NotFoundHandler の設定
// ----------------------------------------
func (m *Mux[T]) SetNotFoundHandler(h Handler[T]) {
	m.notFoundHandler = h
}

func (m *Mux[T]) SetNotFoundHandlerFunc(f func(ctx context.Context, req *Request) (T, error)) {
	m.notFoundHandler = HandlerFunc[T](f)
}

// ----------------------------------------
// ハンドラ登録（Handle, HandleFunc）
// ----------------------------------------
func (m *Mux[T]) HandleFunc(uri string, f func(context.Context, *Request) (T, error)) error {
	return m.Handle(uri, HandlerFunc[T](f))
}

func (m *Mux[T]) Handle(uri string, h Handler[T]) error {
	// URI をパース → 内部用の route[T] 構造体に変換
	r, err := m.parseRoute(uri, h)
	if err != nil {
		return err
	}
	// 登録時に重複や衝突をチェック
	if err := m.checkConflict(r); err != nil {
		return err
	}
	// 追加
	m.routes = append(m.routes, r)
	return nil
}

// ----------------------------------------
// ルックアップ（生 URI から Handler を呼び出す例）
// 実際の利用形態に合わせてメソッド名や署名を変更してください
// ----------------------------------------
func (m *Mux[T]) Lookup(ctx context.Context, rawURI string) (T, error) {
	// パース
	req, parsed, err := m.parseRequest(rawURI)
	if err != nil {
		// クエリの重複や空キーなどは parseRequest 時点でエラーになる
		// → ルートミスマッチと同等扱いで notFoundHandler を呼ぶ
		return m.handleNotFound(ctx, req)
	}

	// 全ルートを走査してマッチ度の高いものを探す
	var candidates []matchedRoute[T]
	for _, rt := range m.routes {
		params, match := m.matchRoute(rt, parsed)
		if match {
			// パラメータを獲得して候補に入れる
			score := calcStaticScore(rt) // 静的セグメント数をスコアとみなす
			candidates = append(candidates, matchedRoute[T]{
				route:  rt,
				params: params,
				score:  score,
			})
		}
	}
	if len(candidates) == 0 {
		// 見つからないので notFoundHandler or ErrNotFound
		return m.handleNotFound(ctx, req)
	}

	// 静的スコア降順にソートし、最もスコアが高いものを選択
	// 「静的ルートがあればそちらを優先」が担保される
	best := pickBestMatch(candidates)

	// Request にパラメータをセット
	for k, v := range best.params {
		req.Params[k] = v
	}

	return best.route.handler.Handle(ctx, req)
}

// ----------------------------------------
// 内部で使用するヘルパー
// ----------------------------------------

// matchedRoute は Lookup 時にマッチした候補を保持するための構造
type matchedRoute[T any] struct {
	route  route[T]
	params map[string]string
	score  int
}

// pickBestMatch は最も score が高いものを返す
func pickBestMatch[T any](candidates []matchedRoute[T]) matchedRoute[T] {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best
}

// calcStaticScore は「静的セグメント数 + ホスト固定なら1、スキームは常に固定なので0加算」など適宜
func calcStaticScore[T any](r route[T]) int {
	score := 0
	// ホストが固定なら +1
	if !r.hostIsParam {
		score++
	}
	// パスの各セグメントが静的なら +1
	for _, seg := range r.pathSegments {
		if !seg.isParam {
			score++
		}
	}
	return score
}

// handleNotFound は notFoundHandler があれば呼び出し、なければ ErrNotFound を返す
func (m *Mux[T]) handleNotFound(ctx context.Context, req *Request) (T, error) {
	var zero T
	if m.notFoundHandler != nil {
		return m.notFoundHandler.Handle(ctx, req)
	}
	return zero, ErrNotFound
}

// ----------------------------------------
// ルート登録時の URI パース
// ----------------------------------------
func (m *Mux[T]) parseRoute(uri string, h Handler[T]) (route[T], error) {
	u, err := url.Parse(uri)
	if err != nil {
		return route[T]{}, fmt.Errorf("invalid URI: %w", err)
	}

	// scheme（小文字化）
	if u.Scheme == "" {
		return route[T]{}, fmt.Errorf("scheme is required")
	}
	scheme := strings.ToLower(u.Scheme)

	// host（小文字化）
	if u.Host == "" {
		return route[T]{}, fmt.Errorf("host is required")
	}
	// 「{param}」かどうか判定
	hostIsParam := false
	hostParamName := ""
	hostLower := strings.ToLower(u.Host)
	if strings.HasPrefix(u.Host, "{") && strings.HasSuffix(u.Host, "}") {
		// 動的ホスト
		hostIsParam = true
		hostParamName = strings.TrimSuffix(strings.TrimPrefix(u.Host, "{"), "}")
		if !isValidParamName(hostParamName) {
			return route[T]{}, fmt.Errorf("invalid host param name: %s", hostParamName)
		}
	} else {
		hostLower = strings.ToLower(u.Host) // 固定ホストは小文字で保持
	}

	// path
	pathSegs, err := parsePath(u.Path)
	if err != nil {
		return route[T]{}, err
	}

	// クエリ（固定キー値のみ）
	q, err := parseQueryForRegistration(u.RawQuery)
	if err != nil {
		return route[T]{}, err
	}

	// 動的パラメータが含まれる場合に名前の重複チェック
	if err := checkParamNameDuplication(hostParamName, pathSegs); err != nil {
		return route[T]{}, err
	}

	r := route[T]{
		scheme:        scheme,
		hostIsParam:   hostIsParam,
		hostParamName: hostParamName,
		host:          hostLower,
		pathSegments:  pathSegs,
		query:         q,
		handler:       h,
	}
	return r, nil
}

// ----------------------------------------
// 登録時の重複・衝突チェック
// ----------------------------------------
func (m *Mux[T]) checkConflict(newRoute route[T]) error {
	for _, rt := range m.routes {
		// 完全に同一の静的ルートが重複していないか（scheme, host固定/param, path, query が同じ形）チェック
		// もしくは動的ルート同士が同一パターンをカバーしていないか
		if isSameCoverage(rt, newRoute) {
			// すでに同じ (または同一カバレッジ) ルートがある
			return fmt.Errorf("conflict route: %v", routeToString(newRoute))
		}
	}
	return nil
}

// isSameCoverage は 2つのルートが「同じマッチ範囲」をカバーするかどうか
// 仕様上:
//   - 静的 vs 静的 が同一なら完全重複
//   - 動的 vs 動的 が同一箇所にあれば同じパターン
//   - 静的 vs 動的 は衝突にならない（ただし同一箇所が静的と動的で両方マッチ可能でも登録エラーにはしない）
func isSameCoverage[T any](a, b route[T]) bool {
	// 1) scheme は小文字化済みなのでそのまま比較
	if a.scheme != b.scheme {
		return false
	}

	// 2) host
	if a.hostIsParam != b.hostIsParam {
		// 片方が動的、もう片方が固定 → 同じカバー範囲ではない
		return false
	} else {
		if a.hostIsParam && b.hostIsParam {
			// 両方動的なら同じカバレッジ
		} else {
			// 両方固定なら値が同じかどうか
			if a.host != b.host {
				return false
			}
		}
	}

	// 3) path
	if len(a.pathSegments) != len(b.pathSegments) {
		return false
	}
	for i := 0; i < len(a.pathSegments); i++ {
		sA, sB := a.pathSegments[i], b.pathSegments[i]
		if sA.isParam != sB.isParam {
			// 片方が静的、片方が動的ならカバー範囲は異なる
			return false
		} else {
			if sA.isParam && sB.isParam {
				// 両方動的なら同じカバレッジ
				// → パラメータ名が違っていても同じカバー範囲
			} else {
				// 両方静的なら文字列が同じかどうか
				if sA.literal != sB.literal {
					return false
				}
			}
		}
	}

	// 4) query
	//   - 同じキー・値のペアが完全一致なら同じカバレッジ
	//   - この場合、「動的クエリ」は仕様で禁止されているので固定値同士の一致チェックのみで十分
	if len(a.query) != len(b.query) {
		return false
	}
	for k, v := range a.query {
		if v2, ok := b.query[k]; !ok || v != v2 {
			return false
		}
	}

	return true
}

// routeToString はデバッグ用
func routeToString[T any](r route[T]) string {
	var sb strings.Builder
	sb.WriteString(r.scheme)
	sb.WriteString("://")
	if r.hostIsParam {
		sb.WriteString("{" + r.hostParamName + "}")
	} else {
		sb.WriteString(r.host)
	}
	sb.WriteString("/")
	for i, seg := range r.pathSegments {
		if i > 0 {
			sb.WriteString("/")
		}
		if seg.isParam {
			sb.WriteString("{" + seg.paramName + "}")
		} else {
			sb.WriteString(seg.literal)
		}
	}
	if len(r.query) > 0 {
		sb.WriteString("?")
		i := 0
		for k, v := range r.query {
			if i > 0 {
				sb.WriteString("&")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			i++
		}
	}
	return sb.String()
}

// ----------------------------------------
// リクエスト用の URI パース
// ----------------------------------------
type parsedURI struct {
	scheme   string
	host     string
	pathSegs []string
	query    map[string]string
	rawQuery map[string]string // 実際に受け取った全クエリ（追加キー含む）
}

// parseRequest は生 URI をパースして Request および内部用構造を返す
func (m *Mux[T]) parseRequest(rawURI string) (*Request, *parsedURI, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return nil, nil, err
	}

	// scheme
	if u.Scheme == "" {
		return nil, nil, fmt.Errorf("scheme is missing in request")
	}
	scheme := strings.ToLower(u.Scheme)

	// host
	if u.Host == "" {
		return nil, nil, fmt.Errorf("host is missing in request")
	}
	host := strings.ToLower(u.Host)

	// path → 正規化（連続スラッシュ/末尾スラッシュなど）
	pathSegs, err := parsePathForRequest(u.Path)
	if err != nil {
		return nil, nil, err
	}

	// クエリをパース → key/value
	reqQuery, err := parseQueryForRequest(u.RawQuery)
	if err != nil {
		return nil, nil, err
	}

	// Request 構造体
	req := &Request{
		Query:  make(map[string]string),
		Params: make(map[string]string),
	}
	// 受け取ったクエリはすべて Request.Query に入れる（マッチの可否は別途判定する）
	for k, v := range reqQuery {
		req.Query[k] = v
	}

	p := &parsedURI{
		scheme:   scheme,
		host:     host,
		pathSegs: pathSegs,
		query:    reqQuery, // 同じもの
		// rawQuery として同じものを持つかどうかはお好み
		rawQuery: reqQuery,
	}

	return req, p, nil
}

// ----------------------------------------
// マッチ判定
// ----------------------------------------
func (m *Mux[T]) matchRoute(rt route[T], parsed *parsedURI) (map[string]string, bool) {
	// 1) scheme
	if rt.scheme != parsed.scheme {
		return nil, false
	}

	params := make(map[string]string)

	// 2) host
	if rt.hostIsParam {
		// 動的ホストとしてパラメータに取り込む
		params[rt.hostParamName] = parsed.host
	} else {
		if rt.host != parsed.host {
			return nil, false
		}
	}

	// 3) path
	if len(rt.pathSegments) != len(parsed.pathSegs) {
		return nil, false
	}
	for i, seg := range rt.pathSegments {
		got := parsed.pathSegs[i]
		if seg.isParam {
			// パラメータをセット
			params[seg.paramName] = got
		} else {
			// 静的セグメントとの一致判定（case-sensitive）
			if seg.literal != got {
				return nil, false
			}
		}
	}

	// 4) query
	//   - 登録されているキーについては、完全一致しているか
	//   - リクエストに余計なキーがあってもよい
	for k, v := range rt.query {
		got, ok := parsed.query[k]
		if !ok {
			return nil, false
		}
		if got != v {
			return nil, false
		}
	}

	return params, true
}

// ----------------------------------------
// path の正規化: 連続スラッシュを1つにまとめ、末尾スラッシュを削除し、""(空) が先頭に出るなら除去
// 例: //users///profile/ → ["users", "profile"]
// 登録時にもリクエスト時にも共通で使う
// ----------------------------------------
func parsePath(p string) ([]pathSegment, error) {
	// スラッシュを正規化する前に先頭/末尾のスラッシュを一旦除去しやすい形に
	trimmed := strings.TrimRight(p, "/")
	// 二重スラッシュを1個にまとめるために Split して再構築
	rawSegs := strings.Split(trimmed, "/")
	var segs []string
	for _, s := range rawSegs {
		if s == "" {
			// 連続するスラッシュの間は無視
			continue
		}
		segs = append(segs, s)
	}

	// pathSegment に変換
	ps := make([]pathSegment, 0, len(segs))
	for _, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			// 動的パラメータ
			paramName := strings.TrimSuffix(strings.TrimPrefix(s, "{"), "}")
			if !isValidParamName(paramName) {
				return nil, fmt.Errorf("invalid path param name: %s", paramName)
			}
			ps = append(ps, pathSegment{
				isParam:   true,
				paramName: paramName,
			})
		} else {
			ps = append(ps, pathSegment{
				isParam: false,
				literal: s,
			})
		}
	}
	return ps, nil
}

// ----------------------------------------
// リクエスト時の path 正規化（こちらは []string で返す）
// 登録時とほぼ同じ処理なのでまとめるならまとめてもOK
// ----------------------------------------
func parsePathForRequest(p string) ([]string, error) {
	trimmed := strings.TrimRight(p, "/")
	rawSegs := strings.Split(trimmed, "/")
	var segs []string
	for _, s := range rawSegs {
		if s == "" {
			continue
		}
		segs = append(segs, s)
	}
	return segs, nil
}

// ----------------------------------------
// クエリパラメータのパース（登録用）
//   - 重複キーがあればエラー
//   - 空キーはエラー
//   - "%xx" は標準ライブラリがデコードしてくれる
//   - "{param}" は禁止（固定値のみ）
//
// ----------------------------------------
func parseQueryForRegistration(q string) (map[string]string, error) {
	if q == "" {
		return map[string]string{}, nil
	}
	values, err := url.ParseQuery(q)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, arr := range values {
		if k == "" {
			return nil, fmt.Errorf("query key cannot be empty")
		}
		if strings.HasPrefix(k, "{") && strings.HasSuffix(k, "}") {
			return nil, fmt.Errorf("dynamic param in query is not allowed: %s", k)
		}
		if len(arr) > 1 {
			// 同じキーに複数値がある → エラー
			return nil, fmt.Errorf("duplicate query key: %s", k)
		}
		result[k] = arr[0]
	}
	return result, nil
}

// ----------------------------------------
// リクエスト時のクエリパース
//   - 重複キーや空キーがあればエラー
//   - 値が複数あるケースもエラー
//
// ----------------------------------------
func parseQueryForRequest(q string) (map[string]string, error) {
	if q == "" {
		return map[string]string{}, nil
	}
	values, err := url.ParseQuery(q)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, arr := range values {
		if k == "" {
			return nil, fmt.Errorf("query key cannot be empty")
		}
		if len(arr) > 1 {
			return nil, fmt.Errorf("duplicate query key: %s", k)
		}
		result[k] = arr[0]
	}
	return result, nil
}

// ----------------------------------------
// パラメータ名のバリデーション
//   - 英数字、ハイフン、アンダースコアのみOK
//
// ----------------------------------------
var paramNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func isValidParamName(name string) bool {
	return name != "" && paramNamePattern.MatchString(name)
}

// ----------------------------------------
// 同一ルート内でパラメータ名が重複していないかチェック
// ホストが {xxx} の場合も考慮
// ----------------------------------------
func checkParamNameDuplication(hostParamName string, pathSegs []pathSegment) error {
	used := make(map[string]bool)
	if hostParamName != "" {
		if used[hostParamName] {
			return fmt.Errorf("param name duplicated in host: %s", hostParamName)
		}
		used[hostParamName] = true
	}
	for _, seg := range pathSegs {
		if seg.isParam {
			if used[seg.paramName] {
				return fmt.Errorf("param name duplicated: %s", seg.paramName)
			}
			used[seg.paramName] = true
		}
	}
	return nil
}
