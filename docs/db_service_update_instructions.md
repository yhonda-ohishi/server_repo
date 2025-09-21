# db_service更新指示書

## 目的
db_serviceのprotoファイルにHTTPアノテーションを追加して、自動的にSwaggerドキュメントが生成されるようにする

## 背景
現在、db_serviceはgRPCサービスとして定義されているが、HTTPアノテーション（`google.api.http`オプション）が含まれていないため、grpc-gatewayによる自動的なSwagger生成ができない。

## 必要な変更

### 1. proto/ryohi.protoの更新

#### 1.1 インポートの追加
```protobuf
import "google/api/annotations.proto";
```

#### 1.2 ETCMeisaiServiceへのHTTPアノテーション追加
```protobuf
service ETCMeisaiService {
  // ETC明細データ作成
  rpc Create(CreateETCMeisaiRequest) returns (ETCMeisaiResponse) {
    option (google.api.http) = {
      post: "/api/v1/db/etc-meisai"
      body: "etc_meisai"
    };
  }

  // ETC明細データ取得
  rpc Get(GetETCMeisaiRequest) returns (ETCMeisaiResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/etc-meisai/{id}"
    };
  }

  // ETC明細データ更新
  rpc Update(UpdateETCMeisaiRequest) returns (ETCMeisaiResponse) {
    option (google.api.http) = {
      put: "/api/v1/db/etc-meisai/{etc_meisai.id}"
      body: "etc_meisai"
    };
  }

  // ETC明細データ削除
  rpc Delete(DeleteETCMeisaiRequest) returns (Empty) {
    option (google.api.http) = {
      delete: "/api/v1/db/etc-meisai/{id}"
    };
  }

  // ETC明細データ一覧取得
  rpc List(ListETCMeisaiRequest) returns (ListETCMeisaiResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/etc-meisai"
    };
  }
}
```

#### 1.3 DTakoUriageKeihiServiceへのHTTPアノテーション追加
```protobuf
service DTakoUriageKeihiService {
  // 経費精算データ作成
  rpc Create(CreateDTakoUriageKeihiRequest) returns (DTakoUriageKeihiResponse) {
    option (google.api.http) = {
      post: "/api/v1/db/dtako-uriage-keihi"
      body: "dtako_uriage_keihi"
    };
  }

  // 経費精算データ取得
  rpc Get(GetDTakoUriageKeihiRequest) returns (DTakoUriageKeihiResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/dtako-uriage-keihi/{srch_id}"
    };
  }

  // 経費精算データ更新
  rpc Update(UpdateDTakoUriageKeihiRequest) returns (DTakoUriageKeihiResponse) {
    option (google.api.http) = {
      put: "/api/v1/db/dtako-uriage-keihi/{dtako_uriage_keihi.srch_id}"
      body: "dtako_uriage_keihi"
    };
  }

  // 経費精算データ削除
  rpc Delete(DeleteDTakoUriageKeihiRequest) returns (Empty) {
    option (google.api.http) = {
      delete: "/api/v1/db/dtako-uriage-keihi/{srch_id}"
    };
  }

  // 経費精算データ一覧取得
  rpc List(ListDTakoUriageKeihiRequest) returns (ListDTakoUriageKeihiResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/dtako-uriage-keihi"
    };
  }
}
```

#### 1.4 DTakoFerryRowsServiceへのHTTPアノテーション追加
```protobuf
service DTakoFerryRowsService {
  // フェリー運行データ作成
  rpc Create(CreateDTakoFerryRowsRequest) returns (DTakoFerryRowsResponse) {
    option (google.api.http) = {
      post: "/api/v1/db/dtako-ferry-rows"
      body: "dtako_ferry_rows"
    };
  }

  // フェリー運行データ取得
  rpc Get(GetDTakoFerryRowsRequest) returns (DTakoFerryRowsResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/dtako-ferry-rows/{id}"
    };
  }

  // フェリー運行データ更新
  rpc Update(UpdateDTakoFerryRowsRequest) returns (DTakoFerryRowsResponse) {
    option (google.api.http) = {
      put: "/api/v1/db/dtako-ferry-rows/{dtako_ferry_rows.id}"
      body: "dtako_ferry_rows"
    };
  }

  // フェリー運行データ削除
  rpc Delete(DeleteDTakoFerryRowsRequest) returns (Empty) {
    option (google.api.http) = {
      delete: "/api/v1/db/dtako-ferry-rows/{id}"
    };
  }

  // フェリー運行データ一覧取得
  rpc List(ListDTakoFerryRowsRequest) returns (ListDTakoFerryRowsResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/dtako-ferry-rows"
    };
  }
}
```

#### 1.5 ETCMeisaiMappingServiceへのHTTPアノテーション追加
```protobuf
service ETCMeisaiMappingService {
  // マッピング作成
  rpc Create(CreateETCMeisaiMappingRequest) returns (ETCMeisaiMappingResponse) {
    option (google.api.http) = {
      post: "/api/v1/db/etc-meisai-mapping"
      body: "etc_meisai_mapping"
    };
  }

  // マッピング取得
  rpc Get(GetETCMeisaiMappingRequest) returns (ETCMeisaiMappingResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/etc-meisai-mapping/{id}"
    };
  }

  // マッピング更新
  rpc Update(UpdateETCMeisaiMappingRequest) returns (ETCMeisaiMappingResponse) {
    option (google.api.http) = {
      put: "/api/v1/db/etc-meisai-mapping/{etc_meisai_mapping.id}"
      body: "etc_meisai_mapping"
    };
  }

  // マッピング削除
  rpc Delete(DeleteETCMeisaiMappingRequest) returns (Empty) {
    option (google.api.http) = {
      delete: "/api/v1/db/etc-meisai-mapping/{id}"
    };
  }

  // マッピング一覧取得
  rpc List(ListETCMeisaiMappingRequest) returns (ListETCMeisaiMappingResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/etc-meisai-mapping"
    };
  }

  // ハッシュからDTakoRowIDを取得
  rpc GetDTakoRowIDByHash(GetDTakoRowIDByHashRequest) returns (GetDTakoRowIDByHashResponse) {
    option (google.api.http) = {
      get: "/api/v1/db/etc-meisai-mapping/by-hash/{etc_meisai_hash}"
    };
  }
}
```

### 2. buf.gen.yamlの更新

db_serviceのbuf.gen.yamlに以下のプラグインを追加:

```yaml
plugins:
  # 既存のプラグイン...

  # OpenAPI/Swagger生成用プラグイン追加
  - plugin: grpc-gateway
    out: .
    opt: paths=source_relative
  - plugin: openapiv2
    out: swagger
    opt:
      - logtostderr=true
      - generate_unbound_methods=true
      - allow_merge=true
      - merge_file_name=db_service
```

### 3. 依存関係の追加

#### 3.1 go.modに追加
```go
require (
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0
    google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97
)
```

#### 3.2 必要なツールのインストール
```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

### 4. コード生成

```bash
# db_serviceディレクトリで実行
buf generate
```

## 期待される効果

1. **自動Swagger生成**: HTTPアノテーションが追加されることで、buf generateコマンドでSwaggerファイルが自動生成される

2. **server-repoでの利用**:
   - db_serviceで生成されたSwaggerファイルをserver-repoから参照可能
   - または、server-repoのbuf.work.yamlにdb_serviceを含めて一括生成

3. **REST APIの一貫性**: HTTPパスとメソッドが明示的に定義され、実装との一貫性が保証される

## 実装優先度

高優先度：この変更により、手動でSwaggerファイルを作成・メンテナンスする必要がなくなり、protoファイルが唯一の真実の源（Single Source of Truth）となる

## 注意事項

1. **後方互換性**: 既存のgRPCクライアントには影響しない（HTTPアノテーションは追加のメタデータ）

2. **パスの命名規則**: REST APIのパス命名規則に従う
   - リソース名は複数形またはケバブケース
   - 例: `/api/v1/db/etc-meisai`, `/api/v1/db/dtako-uriage-keihi`

3. **HTTPメソッドの選択**:
   - GET: 取得・一覧
   - POST: 作成
   - PUT: 更新
   - DELETE: 削除

## テスト方法

1. db_serviceでコード生成を実行
2. 生成されたSwaggerファイルを確認
3. server-repoから参照してSwagger UIで表示確認
4. REST APIエンドポイントの動作確認