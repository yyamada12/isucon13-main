package main

// ISUCON的な参考: https://github.com/isucon/isucon12-qualify/blob/main/webapp/go/isuports.go#L336
// sqlx的な参考: https://jmoiron.github.io/sqlx/

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echolog "github.com/labstack/gommon/log"

	_ "net/http/pprof"
)

type Icon struct {
	data []byte
	hash [32]byte
}

type ReactionsCountMap struct {
	m  map[int64]int64
	mu sync.RWMutex
}

func NewReactionsCountMap() *ReactionsCountMap {
	return &ReactionsCountMap{m: map[int64]int64{}}
}

func (sm *ReactionsCountMap) Add(key int64, count int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = sm.m[key] + count
}

func (sm *ReactionsCountMap) Get(key int64) int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.m[key]
}

func (sm *ReactionsCountMap) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m = map[int64]int64{}
}

type SyncListMap[T any] struct {
	m  map[int64][]T
	mu sync.RWMutex
}

func NewSyncListMap[T any]() *SyncListMap[T] {
	return &SyncListMap[T]{m: map[int64][]T{}}
}

func (sm *SyncListMap[T]) Add(key int64, value T) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = append(sm.m[key], value)
}

func (sm *SyncListMap[T]) Get(key int64) []T {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.m[key]
}

func (sm *SyncListMap[T]) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m = map[int64][]T{}
}

type SyncMap[T any] struct {
	m  map[int64]*T
	mu sync.RWMutex
}

func NewSyncMap[T any]() *SyncMap[T] {
	return &SyncMap[T]{m: map[int64]*T{}}
}

func (sm *SyncMap[T]) Add(key int64, value T) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = &value
}

func (sm *SyncMap[T]) Get(key int64) *T {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.m[key]
}

func (sm *SyncMap[T]) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m = map[int64]*T{}
}

var fallbackImageHash [32]byte
var iconMap = NewSyncMap[Icon]()
var userMap = NewSyncMap[UserModel]()
var themeMap = NewSyncMap[ThemeModel]()
var tagsMap = NewSyncMap[Tag]()
var livestreamTagsMap = NewSyncListMap[Tag]()
var userIDByLiveStreamMap = NewSyncMap[int64]()
var reactionsCountMap = NewReactionsCountMap()

func initCache() {
	loadFllbackImageHash()
	iconMap.Clear()
	userMap.Clear()
	loadUser()
	themeMap.Clear()
	loadTheme()
	livestreamTagsMap.Clear()
	loadLivestreamTags()
	tagsMap.Clear()
	loadTags()
	userIDByLiveStreamMap.Clear()
	loadUserIDByLiveStream()
	reactionsCountMap.Clear()
	loadReactionsCount()
}

func loadFllbackImageHash() {
	image, err := os.ReadFile(fallbackImage)
	if err != nil {
		log.Printf("failed to load fallback image: %+v", err)
		return
	}
	fallbackImageHash = sha256.Sum256(image)
}

func loadUser() {
	users := []UserModel{}
	if err := dbConn.Select(&users, "SELECT * FROM users"); err != nil {
		log.Printf("failed to load users: %+v", err)
		return
	}
	for _, u := range users {
		userMap.Add(u.ID, u)
	}
}

func loadTheme() {
	themes := []ThemeModel{}
	if err := dbConn.Select(&themes, "SELECT * FROM themes"); err != nil {
		log.Printf("failed to load themes: %+v", err)
		return
	}
	for _, t := range themes {
		themeMap.Add(t.UserID, t)
	}
}

type LiveTag struct {
	ID           int64  `db:"id" json:"id"`
	LivestreamID int64  `db:"livestream_id" json:"livestream_id"`
	TagID        int64  `db:"tag_id" json:"tag_id"`
	TagName      string `db:"tag_name" json:"tag_name"`
}

func loadLivestreamTags() {
	var tags []*LiveTag
	if err := dbConn.Select(&tags, "SELECT a.*, b.name as tag_name FROM livestream_tags a JOIN tags b ON a.tag_id = b.id"); err != nil {
		log.Fatalf("failed to load livestream_tags: %+v", err)
		return
	}
	for _, t := range tags {
		livestreamTagsMap.Add(t.LivestreamID, Tag{t.TagID, t.TagName})
	}
}

func loadTags() {
	tags := []Tag{}
	if err := dbConn.Select(&tags, "SELECT * FROM tags"); err != nil {
		log.Printf("failed to load tags: %+v", err)
		return
	}
	for _, t := range tags {
		tagsMap.Add(t.ID, t)
	}
}

func loadUserIDByLiveStream() {
	livestreams := []LivestreamModel{}
	if err := dbConn.Select(&livestreams, "SELECT * FROM livestreams"); err != nil {
		log.Printf("failed to load livestreams: %+v", err)
		return
	}
	for _, l := range livestreams {
		userIDByLiveStreamMap.Add(l.ID, l.UserID)
	}
}

func loadReactionsCount() {
	reactions := []ReactionModel{}
	if err := dbConn.Select(&reactions, "SELECT * FROM reactions"); err != nil {
		log.Printf("failed to load reactions: %+v", err)
		return
	}
	for _, r := range reactions {
		userID := userIDByLiveStreamMap.Get(r.LivestreamID)
		reactionsCountMap.Add(*userID, 1)
	}
}

const (
	listenPort                     = 8080
	powerDNSSubdomainAddressEnvKey = "ISUCON13_POWERDNS_SUBDOMAIN_ADDRESS"
)

var (
	powerDNSSubdomainAddress string
	dbConn                   *sqlx.DB
	secret                   = []byte("isucon13_session_cookiestore_defaultsecret")
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if secretKey, ok := os.LookupEnv("ISUCON13_SESSION_SECRETKEY"); ok {
		secret = []byte(secretKey)
	}
}

type InitializeResponse struct {
	Language string `json:"language"`
}

func connectDB(logger echo.Logger) (*sqlx.DB, error) {
	const (
		networkTypeEnvKey = "ISUCON13_MYSQL_DIALCONFIG_NET"
		addrEnvKey        = "ISUCON13_MYSQL_DIALCONFIG_ADDRESS"
		portEnvKey        = "ISUCON13_MYSQL_DIALCONFIG_PORT"
		userEnvKey        = "ISUCON13_MYSQL_DIALCONFIG_USER"
		passwordEnvKey    = "ISUCON13_MYSQL_DIALCONFIG_PASSWORD"
		dbNameEnvKey      = "ISUCON13_MYSQL_DIALCONFIG_DATABASE"
		parseTimeEnvKey   = "ISUCON13_MYSQL_DIALCONFIG_PARSETIME"
	)

	conf := mysql.NewConfig()

	// 環境変数がセットされていなかった場合でも一旦動かせるように、デフォルト値を入れておく
	// この挙動を変更して、エラーを出すようにしてもいいかもしれない
	conf.Net = "tcp"
	conf.Addr = net.JoinHostPort("127.0.0.1", "3306")
	conf.User = "isucon"
	conf.Passwd = "isucon"
	conf.DBName = "isupipe"
	conf.ParseTime = true
	conf.InterpolateParams = true

	if v, ok := os.LookupEnv(networkTypeEnvKey); ok {
		conf.Net = v
	}
	if addr, ok := os.LookupEnv(addrEnvKey); ok {
		if port, ok2 := os.LookupEnv(portEnvKey); ok2 {
			conf.Addr = net.JoinHostPort(addr, port)
		} else {
			conf.Addr = net.JoinHostPort(addr, "3306")
		}
	}
	if v, ok := os.LookupEnv(userEnvKey); ok {
		conf.User = v
	}
	if v, ok := os.LookupEnv(passwordEnvKey); ok {
		conf.Passwd = v
	}
	if v, ok := os.LookupEnv(dbNameEnvKey); ok {
		conf.DBName = v
	}
	if v, ok := os.LookupEnv(parseTimeEnvKey); ok {
		parseTime, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse environment variable '%s' as bool: %+v", parseTimeEnvKey, err)
		}
		conf.ParseTime = parseTime
	}

	db, err := sqlx.Open("mysql", conf.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)

	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Print(err)
		time.Sleep(time.Second * 2)
	}
	log.Print("DB ready!")

	// if err := db.Ping(); err != nil {
	// 	return nil, err
	// }

	return db, nil
}

func initializeHandler(c echo.Context) error {
	if out, err := exec.Command("../sql/init.sh").CombinedOutput(); err != nil {
		c.Logger().Warnf("init.sh failed with err=%s", string(out))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to initialize: "+err.Error())
	}

	// if err := os.RemoveAll("/home/isucon/webapp/icon/"); err != nil {
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove directory: "+err.Error())
	// }
	// if err := os.Mkdir("/home/isucon/webapp/icon/", 0755); err != nil {
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "failed to create directory: "+err.Error())
	// }

	req, err := http.NewRequest("POST", "http://192.168.0.13:8080/api/initialize_dns", nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create request: "+err.Error())
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send request: "+err.Error())
	}
	defer resp.Body.Close()

	initCache()

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	return c.JSON(http.StatusOK, InitializeResponse{
		Language: "golang",
	})
}

func initializeDNSHandler(c echo.Context) error {
	if out, err := exec.Command("../pdns/init_zone.sh").CombinedOutput(); err != nil {
		c.Logger().Warnf("init_zone.sh failed with err=%s", string(out))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to initialize: "+err.Error())
	}

	// if err := os.RemoveAll("/home/isucon/webapp/icon/"); err != nil {
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove directory: "+err.Error())
	// }
	// if err := os.Mkdir("/home/isucon/webapp/icon/", 0755); err != nil {
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "failed to create directory: "+err.Error())
	// }

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	return c.JSON(http.StatusOK, InitializeResponse{
		Language: "golang",
	})
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	e := echo.New()
	e.Debug = true
	e.Logger.SetLevel(echolog.DEBUG)
	e.Use(middleware.Logger())
	cookieStore := sessions.NewCookieStore(secret)
	cookieStore.Options.Domain = "*.u.isucon.dev"
	e.Use(session.Middleware(cookieStore))
	// e.Use(middleware.Recover())

	// 初期化
	e.POST("/api/initialize", initializeHandler)
	e.POST("/api/initialize_dns", initializeDNSHandler)

	// top
	e.GET("/api/tag", getTagHandler)
	e.GET("/api/user/:username/theme", getStreamerThemeHandler)

	// livestream
	// reserve livestream
	e.POST("/api/livestream/reservation", reserveLivestreamHandler)
	// list livestream
	e.GET("/api/livestream/search", searchLivestreamsHandler)
	e.GET("/api/livestream", getMyLivestreamsHandler)
	e.GET("/api/user/:username/livestream", getUserLivestreamsHandler)
	// get livestream
	e.GET("/api/livestream/:livestream_id", getLivestreamHandler)
	// get polling livecomment timeline
	e.GET("/api/livestream/:livestream_id/livecomment", getLivecommentsHandler)
	// ライブコメント投稿
	e.POST("/api/livestream/:livestream_id/livecomment", postLivecommentHandler)
	e.POST("/api/livestream/:livestream_id/reaction", postReactionHandler)
	e.GET("/api/livestream/:livestream_id/reaction", getReactionsHandler)

	// (配信者向け)ライブコメントの報告一覧取得API
	e.GET("/api/livestream/:livestream_id/report", getLivecommentReportsHandler)
	e.GET("/api/livestream/:livestream_id/ngwords", getNgwords)
	// ライブコメント報告
	e.POST("/api/livestream/:livestream_id/livecomment/:livecomment_id/report", reportLivecommentHandler)
	// 配信者によるモデレーション (NGワード登録)
	e.POST("/api/livestream/:livestream_id/moderate", moderateHandler)

	// livestream_viewersにINSERTするため必要
	// ユーザ視聴開始 (viewer)
	e.POST("/api/livestream/:livestream_id/enter", enterLivestreamHandler)
	// ユーザ視聴終了 (viewer)
	e.DELETE("/api/livestream/:livestream_id/exit", exitLivestreamHandler)

	// user
	e.POST("/api/register", registerHandler)
	e.POST("/api/login", loginHandler)
	e.GET("/api/user/me", getMeHandler)
	// フロントエンドで、配信予約のコラボレーターを指定する際に必要
	e.GET("/api/user/:username", getUserHandler)
	e.GET("/api/user/:username/statistics", getUserStatisticsHandler)
	e.GET("/api/user/:username/icon", getIconHandler)
	e.POST("/api/icon", postIconHandler)

	// stats
	// ライブ配信統計情報
	e.GET("/api/livestream/:livestream_id/statistics", getLivestreamStatisticsHandler)

	// 課金情報
	e.GET("/api/payment", GetPaymentResult)

	e.HTTPErrorHandler = errorResponseHandler

	// DB接続
	conn, err := connectDB(e.Logger)
	if err != nil {
		e.Logger.Errorf("failed to connect db: %v", err)
		os.Exit(1)
	}
	defer conn.Close()
	dbConn = conn

	initCache()

	subdomainAddr, ok := os.LookupEnv(powerDNSSubdomainAddressEnvKey)
	if !ok {
		e.Logger.Errorf("environ %s must be provided", powerDNSSubdomainAddressEnvKey)
		os.Exit(1)
	}
	powerDNSSubdomainAddress = subdomainAddr

	// HTTPサーバ起動
	listenAddr := net.JoinHostPort("", strconv.Itoa(listenPort))
	if err := e.Start(listenAddr); err != nil {
		e.Logger.Errorf("failed to start HTTP server: %v", err)
		os.Exit(1)
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func errorResponseHandler(err error, c echo.Context) {
	c.Logger().Errorf("error at %s: %+v", c.Path(), err)
	if he, ok := err.(*echo.HTTPError); ok {
		if e := c.JSON(he.Code, &ErrorResponse{Error: err.Error()}); e != nil {
			c.Logger().Errorf("%+v", e)
		}
		return
	}

	if e := c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: err.Error()}); e != nil {
		c.Logger().Errorf("%+v", e)
	}
}
