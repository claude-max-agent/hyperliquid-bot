# Hyperliquid HFT Bot

Hyperliquid取引所向け高頻度取引ボット。Go言語とクリーンアーキテクチャで構築。

## 機能

- **AIシグナル取引戦略**: 複数データソースからのシグナルを統合した自動取引
- **リスク管理**: ポジションサイズ制限、最大レバレッジ、ドローダウン監視
- **リアルタイムデータ**: WebSocket接続による低遅延マーケットデータ
- **テストネット対応**: 本番環境とテストネットの切り替え可能

## アーキテクチャ

```
cmd/bot/               # エントリーポイント
internal/
├── domain/            # ビジネスロジック（取引所非依存）
│   ├── entity/        # コアエンティティ（Order, Position等）
│   ├── repository/    # リポジトリインターフェース
│   └── service/       # ドメインサービス（Strategy等）
├── usecase/           # アプリケーションユースケース
├── adapter/           # インターフェースアダプター
│   ├── controller/    # HTTP/CLIコントローラー
│   ├── gateway/       # 外部サービスインターフェース
│   └── presenter/     # 出力フォーマッター
└── infrastructure/    # フレームワーク＆ドライバー
    ├── hyperliquid/   # Hyperliquid実装
    ├── coinglass/     # CoinGlass API
    ├── whalealert/    # Whale Alert API
    ├── lunarcrush/    # LunarCrush API
    ├── macro/         # マクロ指標（FedWatch, Trading Economics）
    ├── signal/        # シグナルプロバイダー
    ├── config/        # 設定ローダー
    └── logger/        # ロギング
```

## クイックスタート

### 必要環境

- Go 1.22以上
- Make

### セットアップ

```bash
# リポジトリをクローン
git clone https://github.com/claude-max-agent/hyperliquid-bot.git
cd hyperliquid-bot

# 依存関係をインストール
make deps

# 設定ファイルをコピー
cp config/config.example.yaml config/config.yaml
cp .env.example .env

# .envファイルを編集してAPI認証情報を設定
# 詳細は下記「APIキー設定」セクションを参照

# ビルド
make build

# 実行
make run
```

## APIキー設定

### 必須: Hyperliquid API

Hyperliquidでの取引に必要です。

| 環境変数 | 説明 |
|----------|------|
| `EXCHANGE_API_KEY` | HyperliquidのAPIキー |
| `EXCHANGE_API_SECRET` | HyperliquidのAPIシークレット |

**APIキー取得方法:**
1. [Hyperliquid](https://app.hyperliquid.xyz/) にアクセス
2. ウォレットを接続
3. 右上メニューから「API」を選択
4. 新しいAPIキーを生成

**テストネット:**
- テストネットURL: https://app.hyperliquid-testnet.xyz/
- テストネットAPI: https://api.hyperliquid-testnet.xyz
- `EXCHANGE_TESTNET=true` で有効化

### オプション: AIシグナル取引用データソース

AIシグナル取引戦略を使用する場合、以下のAPIキーを設定してください。

#### CoinGlass（デリバティブデータ）

Funding Rate、OI、ロング/ショート比率、清算データを提供。

| 環境変数 | 説明 |
|----------|------|
| `COINGLASS_API_KEY` | CoinGlass APIキー |

**APIキー取得:**
1. [CoinGlass API](https://www.coinglass.com/CryptoApi) にアクセス
2. アカウント登録
3. プランを選択（無料プランあり）
4. APIキーを生成

#### Whale Alert（大口取引監視）

大口のブロックチェーン取引をリアルタイム監視。

| 環境変数 | 説明 |
|----------|------|
| `WHALE_ALERT_API_KEY` | Whale Alert APIキー |
| `WHALE_ALERT_MIN_VALUE` | 最小取引額（USD、デフォルト: 500000） |

**APIキー取得:**
1. [Whale Alert](https://whale-alert.io/) にアクセス
2. 「API」メニューからサインアップ
3. プランを選択（無料プランあり）
4. APIキーを取得

#### LunarCrush（ソーシャルセンチメント）

SNSセンチメント分析、Galaxy Score、トレンドトピックを提供。

| 環境変数 | 説明 |
|----------|------|
| `LUNARCRUSH_API_KEY` | LunarCrush APIキー |

**APIキー取得:**
1. [LunarCrush Developers](https://lunarcrush.com/developers/api/endpoints) にアクセス
2. アカウント作成
3. Developer PortalでAPIキーを生成

#### CME FedWatch（Fed金利予想）

FOMC会合での利上げ/利下げ確率を提供。

| 環境変数 | 説明 |
|----------|------|
| `FEDWATCH_API_KEY` | FedWatch APIキー |

**APIキー取得:**
1. [CME Group FedWatch API](https://www.cmegroup.com/market-data/market-data-api/fedwatch-api.html) にアクセス
2. APIアクセスを申請（月額$25〜）
3. 承認後、APIキーを取得

#### Trading Economics（経済指標）

CPI、GDP、失業率などの経済指標と経済カレンダーを提供。

| 環境変数 | 説明 |
|----------|------|
| `TRADING_ECONOMICS_API_KEY` | Trading Economics APIキー |

**APIキー取得:**
1. [Trading Economics API](https://tradingeconomics.com/api/) にアクセス
2. アカウント登録
3. プランを選択
4. APIキーを取得

## 設定

設定はYAMLファイルと環境変数の両方をサポート。環境変数が優先されます。

### YAML設定

詳細は `config/config.example.yaml` を参照。

### 主要な環境変数

| 変数 | 説明 | デフォルト |
|------|------|-----------|
| `EXCHANGE_TESTNET` | テストネット使用 | `true` |
| `LOG_LEVEL` | ログレベル | `info` |
| `RISK_MAX_POSITION_SIZE` | 最大ポジションサイズ | `1.0` |
| `RISK_MAX_LEVERAGE` | 最大レバレッジ | `3.0` |

## 開発

```bash
# テスト実行
make test

# リンター実行
make lint

# コードフォーマット
make fmt

# 開発ツールインストール
make tools
```

## 戦略インターフェース

カスタム取引戦略を作成するには `Strategy` インターフェースを実装:

```go
type Strategy interface {
    Name() string
    Init(ctx context.Context, config map[string]interface{}) error
    OnTick(ctx context.Context, state *MarketState) ([]*Signal, error)
    OnOrderUpdate(ctx context.Context, order *entity.Order) error
    OnPositionUpdate(ctx context.Context, position *entity.Position) error
    Stop(ctx context.Context) error
}
```

### 組み込み戦略

| 戦略名 | 説明 |
|--------|------|
| `mean_reversion` | 平均回帰戦略（ボリンジャーバンド的アプローチ） |
| `ai_signal` | AIシグナル戦略（複数データソース統合） |

## データフロー（AIシグナル戦略）

```
[5データソース] → [Signal Provider] → [MarketSignal] → [AI Strategy] → [取引判断]

データソース:
├── CoinGlass     : FR, OI, L/S比率, 清算
├── Whale Alert   : 取引所入出金
├── LunarCrush    : ソーシャルセンチメント
├── CME FedWatch  : 金利確率
└── Trading Econ. : CPI, GDP, 失業率
```

## ライセンス

MIT

## 注意事項

- **本番環境での使用は自己責任でお願いします**
- テストネットで十分なテストを行ってから本番環境で使用してください
- APIキーは安全に管理し、公開リポジトリにコミットしないでください
