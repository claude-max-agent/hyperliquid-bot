# ファンディングレートアービトラージ 実践ガイド

作成日: 2026-02-04
担当: worker3

---

## 1. 概要

### 1.1 ファンディングレートとは

無期限先物（Perpetual Futures）では、現物価格との乖離を防ぐために「ファンディングレート」という仕組みがあります。

```
ファンディングレート > 0 (正): ロングがショートに支払う
ファンディングレート < 0 (負): ショートがロングに支払う
```

### 1.2 アービトラージの原理

同一資産で**ロングとショートを両建て**することで、価格変動リスクをゼロにしつつ、ファンディングレートの差額を受け取ります。

```
例: BTC ファンディングレート +0.01%（8時間ごと）

現物でBTCを購入（または低レートの取引所でロング）
先物でBTCをショート（高レートの取引所）

→ 価格変動は相殺
→ ファンディングレート分を受け取り
→ 年利換算: 0.01% × 3回/日 × 365日 = 約10.95%
```

---

## 2. 必要なツール・環境

### 2.1 取引所アカウント

| 取引所 | 特徴 | KYC | 日本人利用 |
|--------|------|-----|-----------|
| **Hyperliquid** | DEX、1時間ごとファンディング | なし | 可能 |
| Binance | 最大流動性、8時間ごと | 必要 | グレー |
| Bybit | 高流動性、8時間ごと | 必要 | グレー |
| OKX | 多機能、8時間ごと | 必要 | グレー |

**推奨組み合わせ**: Hyperliquid + Bybit（または OKX）

### 2.2 監視ツール

| ツール | URL | 機能 |
|--------|-----|------|
| **CoinGlass** | https://www.coinglass.com/FundingRate | リアルタイムレート比較 |
| **Loris Tools** | https://loris.tools/ | アービトラージスクリーナー |
| Binance Arbitrage Data | https://www.binance.com/en/futures/funding-history/perpetual/arbitrage-data | 公式データ |

### 2.3 必要資金

| 項目 | 最小 | 推奨 |
|------|------|------|
| 初期資金 | $1,000 | $5,000〜 |
| 各取引所配分 | 50%ずつ | 50%ずつ |
| レバレッジ | 1x〜2x | 1x（推奨） |

### 2.4 技術環境（自動化する場合）

```bash
# Python環境
pip install hyperliquid-python-sdk
pip install python-binance  # または ccxt

# 監視用
pip install pandas requests websocket-client
```

---

## 3. 具体的な実行ステップ

### Step 1: ファンディングレートの確認

```
1. CoinGlass (https://www.coinglass.com/FundingRate) にアクセス
2. 「Predicted」列で次回のファンディングレートを確認
3. 取引所間の差が大きい通貨を探す

良い条件:
- レート差が 0.01% 以上
- 安定して正または負が続いている
- 流動性が十分ある（BTC, ETH, SOL等）
```

### Step 2: ポジション構築

#### パターンA: 現物 + 先物ショート（シンプル）

```
【取引所A: 現物】
- BTC を $10,000 分購入

【取引所B: 先物】
- BTC を $10,000 分ショート（1x レバレッジ）

→ 価格が上がっても下がっても損益はゼロ
→ ファンディングレートが正なら、ショート側が受け取り
```

#### パターンB: 先物ロング + 先物ショート（クロス取引所）

```
【Hyperliquid: 先物ロング】
- BTC を $10,000 分ロング
- ファンディング: 1時間ごと

【Bybit: 先物ショート】
- BTC を $10,000 分ショート
- ファンディング: 8時間ごと

→ レート差を利用
```

### Step 3: 監視と管理

```python
# 簡易監視スクリプト例
import requests

def get_funding_rates():
    # CoinGlass API（例）
    url = "https://open-api.coinglass.com/public/v2/funding"
    response = requests.get(url)
    return response.json()

def check_arbitrage_opportunity(symbol="BTC"):
    rates = get_funding_rates()
    # 取引所間のレート差を計算
    # 閾値を超えたらアラート
    pass
```

### Step 4: ポジションクローズ

```
クローズのタイミング:
1. ファンディングレートが逆転した時
2. レート差が縮小した時（0.005%以下など）
3. 目標利益に達した時

手順:
1. 両方のポジションを同時にクローズ
2. スリッページを最小化するため成行注文
3. 資金を引き出しまたは次の機会を待つ
```

---

## 4. リスク管理

### 4.1 主要リスクと対策

| リスク | 説明 | 対策 |
|--------|------|------|
| **清算リスク** | レバレッジ使用時に急変動で清算 | 1xレバレッジ推奨、証拠金維持率監視 |
| **レート逆転** | ファンディングレートが反転 | 自動監視、損切りライン設定 |
| **執行リスク** | 同時決済できない | 流動性の高い通貨、成行注文 |
| **取引所リスク** | 取引所の障害・破綻 | 分散、過度な資金集中を避ける |
| **規制リスク** | 日本居住者への規制 | 法的助言を得る、自己責任で判断 |

### 4.2 損切りルール

```
1. ファンディングレートが3回連続で逆方向 → クローズ検討
2. 累積損失が元本の2%を超えた → 強制クローズ
3. 取引所の異常（出金停止等） → 即座にクローズ
```

### 4.3 資金管理

```
推奨ポートフォリオ配分:
- アービトラージ用: 総資産の30-50%
- 予備資金（証拠金追加用）: 20%
- 他の投資/現金: 30-50%
```

---

## 5. 期待リターン

### 5.1 シミュレーション

```
前提条件:
- 元本: $10,000
- 平均ファンディングレート差: 0.01%（8時間）
- 取引日数: 365日
- 手数料: 往復0.1%

計算:
年間ファンディング収入 = $10,000 × 0.01% × 3回/日 × 365日
                      = $10,950

手数料（2回のポジション構築・解消）= $10,000 × 0.1% × 4回
                                   = $40

純利益 = $10,950 - $40 = $10,910
年利 = 約109%（理想的な場合）

現実的な見積もり（レート変動考慮）:
年利 = 10-30%
```

### 5.2 実績データ

```
OKXアービトラージBot実績（バックテスト）:
- APY: 4.39% 〜 9.46%
- リスク: 低

Binance Delta-Neutral:
- APY: 5% 〜 15%
- リスク: 低〜中
```

---

## 6. 自動化（上級者向け）

### 6.1 Hyperliquid Python SDK

```python
from hyperliquid.info import Info
from hyperliquid.exchange import Exchange
from hyperliquid.utils import constants

# 接続設定
info = Info(constants.MAINNET_API_URL, skip_ws=True)
exchange = Exchange(wallet, constants.MAINNET_API_URL)

# ファンディングレート取得
def get_funding_rate(symbol):
    meta = info.meta()
    for asset in meta['universe']:
        if asset['name'] == symbol:
            return asset['funding']
    return None

# ポジション作成
def open_short(symbol, size):
    order = exchange.order(
        symbol,
        is_buy=False,
        sz=size,
        limit_px=None,  # 成行
        order_type={"market": {}}
    )
    return order
```

### 6.2 監視Bot構成

```
┌─────────────────────────────────────┐
│         Funding Rate Monitor        │
│  - CoinGlass/取引所APIからレート取得   │
│  - 閾値超えでアラート                 │
└─────────────┬───────────────────────┘
              │
┌─────────────▼───────────────────────┐
│        Position Manager             │
│  - Hyperliquid: ロング/ショート       │
│  - CEX: 反対ポジション                │
│  - 同時執行                          │
└─────────────┬───────────────────────┘
              │
┌─────────────▼───────────────────────┐
│         Risk Monitor                │
│  - 証拠金維持率監視                   │
│  - 損切り自動執行                     │
│  - Discord/Telegram通知             │
└─────────────────────────────────────┘
```

---

## 7. チェックリスト

### 開始前

- [ ] 取引所アカウント作成（Hyperliquid + CEX）
- [ ] 資金入金（各取引所に分配）
- [ ] 監視ツールの設定
- [ ] リスク許容度の確認
- [ ] 損切りルールの決定

### 実行中

- [ ] 毎日ファンディングレートを確認
- [ ] ポジションサイズのバランス確認
- [ ] 証拠金維持率の確認
- [ ] 累積損益の記録

### 緊急時

- [ ] 両ポジションの即座クローズ手順を把握
- [ ] 取引所サポートの連絡先確認
- [ ] 予備資金の準備

---

## 8. 参考リソース

### ドキュメント
- [Hyperliquid Docs](https://hyperliquid.gitbook.io/hyperliquid-docs/)
- [Binance Funding Rate](https://www.binance.com/en/futures/funding-history)
- [OKX Funding Rate](https://www.okx.com/learn/funding-rates-perpetual-futures-strategies)

### ツール
- [CoinGlass Funding Rate](https://www.coinglass.com/FundingRate)
- [Loris Tools Screener](https://loris.tools/)
- [Hyperliquid Python SDK](https://github.com/hyperliquid-dex/hyperliquid-python-sdk)

### 解説記事
- [Funding Rate Arbitrage Explained](https://madeinark.org/funding-rate-arbitrage-and-perpetual-futures-the-hidden-yield-strategy-in-cryptocurrency-derivatives-markets/)
- [Hyperliquid vs CEX Arbitrage](https://news.chainspot.io/2025/11/18/basis-funding-cross-venue-arbitrage-trading-hyperliquid-vs-cex-and-l2-dexs/)

---

## 注意事項

**免責事項**: 本レポートは情報提供のみを目的としており、投資助言ではありません。仮想通貨取引は高いリスクを伴います。日本居住者は関連法規を確認し、自己責任で判断してください。

---

_作成: worker3 | raspy-claude-agent_
