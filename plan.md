# Claude Code 指示書 - server-repo実装

## プロジェクト概要
gRPC-Firstアーキテクチャのserver-repoを実装します。このリポジトリは、gRPCサーバーから複数プロトコル（REST、gRPC-Web、JSON-RPC2）への変換を行うGatewayサーバーです。

## 実装対象
- **リポジトリ**: `server-repo`
- **役割**: Multi-Protocol Gateway + 依存性注入 + OpenAPI Swagger提供
- **技術スタック**: Go + Fiber + grpc-gateway + bufconn

## アーキテクチャ要件

### 1. 単一プロセス vs 分離プロセス対応
環境変数`DEPLOYMENT_MODE`で動作モードを切り替え:
- `single`: 全てを単一プロセス内で実行（本番推奨）
- `separate`: 各層を別プロセスで実行（開発・テスト用）

### 2. 提供プロトコル
- **gRPC**: 高性能バイナリプロトコル
- **REST API**: grpc-gatewayによる自動変換
- **gRPC-Web**: ブラウザ対応
- **JSON-RPC2**: 軽量JSONプロトコル
- **OpenAPI**: Swagger UI提供

## ディレクトリ構造
```
server-repo/
├── cmd/
│   └── server/
│       └── main.go                 # エントリーポイント
├── internal/
│   ├── gateway/
│   │   ├── gateway.go             # マルチプロトコルGateway
│   │   ├── grpc_web.go           # gRPC-Web対応
│   │   └── jsonrpc.go            # JSON-RPC2実装
│   ├── client/
│   │   ├── bufconn.go            # インメモリgRPCクライアント
│   │   └── network.go            # ネットワークgRPCクライアント
│   ├── config/
│   │   └── config.go             # 設定管理
│   └── health/
│       └── health.go             # ヘルスチェック
├── proto/                         # Protocol Buffers定義（handlers-repoからコピー）
│   └── user.proto
├── swagger/                       # 自動生成されるOpenAPI仕様
│   └── user.swagger.json
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 実装要件

### 1. main.go（エントリーポイント）
```go
func main() {
    mode := os.Getenv("DEPLOYMENT_MODE")
    if mode == "" {
        mode = "single" // デフォルト
    }
    
    switch mode {
    case "single":
        runSingleProcess()
    case "separate":
        runSeparateProcesses()
    default:
        log.Fatal("Invalid DEPLOYMENT_MODE. Use 'single' or 'separate'")
    }
}
```

### 2. 単一プロセスモード実装
- database-repoとhandlers-repoを同一プロセス内で起動
- bufconnを使用したメモリ内gRPC通信
- 全ての依存性を内部で解決

### 3. 分離プロセスモード実装
- 外部gRPCサーバーへのネットワーク接続
- 環境変数による接続先設定
- 接続エラーハンドリング

### 4. Gateway機能
- grpc-gatewayによるREST API自動変換
- gRPC-Web対応（Envoyプロキシ不要）
- カスタムJSON-RPC2実装
- OpenAPI/Swagger UI提供

### 5. 設定管理
環境変数:
```bash
# 動作モード
DEPLOYMENT_MODE=single|separate

# 分離モード用（separateの場合必須）
DATABASE_GRPC_URL=localhost:50051
HANDLERS_GRPC_URL=localhost:50052

# データベース設定
DATABASE_URL=user:pass@tcp(localhost:3306)/dbname

# サーバー設定
SERVER_PORT=8080
LOG_LEVEL=info
```

## 依存関係
```go
// go.mod必須パッケージ
require (
    github.com/gofiber/fiber/v2 v2.50.0
    google.golang.org/grpc v1.59.0
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1
    google.golang.org/protobuf v1.31.0
    
    // 内部依存（実際の実装では相対パスまたはモジュール名）
    // github.com/yhonda-ohishi/db-handler-server/database-repo
    // github.com/yhonda-ohishi/db-handler-server/handlers-repo
)
```

## API仕様
### REST API エンドポイント
 https://github.com/yhonda-ohishi/etc_meisai
 のエンドポイントを開放
### その他エンドポイント
- `GET /docs` - Swagger UI
- `GET /swagger/user.swagger.json` - OpenAPI仕様
- `GET /health` - ヘルスチェック
- `/grpc-web/*` - gRPC-Webエンドポイント
- `POST /jsonrpc` - JSON-RPC2エンドポイント

## テスト要件
1. **単体テスト**: 各コンポーネントの独立テスト
2. **統合テスト**: プロトコル変換テスト
3. **E2Eテスト**: 全プロトコルの動作確認

## 実装手順
1. プロジェクト初期化（go mod init）
2. 依存関係追加
3. Protocol Buffers定義とコード生成
4. 設定管理実装
5. gRPCクライアント実装（bufconn + network）
6. Gateway実装
7. main.go実装
8. テスト実装
9. Docker化

## 期待される動作
```bash
# 単一プロセスモード起動
export DEPLOYMENT_MODE=single
go run cmd/server/main.go

# 各プロトコルテスト
curl http://localhost:8080/api/v1/users/1    # REST
curl http://localhost:8080/docs              # Swagger UI
grpcurl -plaintext localhost:8080 UserService/GetUser  # gRPC
```

## 注意事項
- エラーハンドリングを適切に実装
- ログ出力を充実させる
- グレースフルシャットダウン対応
- メトリクス取得準備（将来拡張用）
- セキュリティヘッダー設定
- CORS対応

このserver-repoが完成すると、gRPCで定義したAPIが自動的に複数プロトコルで利用可能になり、OpenAPI仕様も自動生成されます。