# softether-tui 開発ログ (app_devlog.md)

## 2026-07-30: Docker実環境による動的接続検証と課題修正

### 1. 接続プロファイルのデフォルト登録とパスワード対話フローの拡充
- **デフォルト接続先の追加**:
  - `internal/config/profile.go` を拡張し、プロファイル保存ファイル (`profiles.yaml`) が未存在または空の場合に、自動的に `localhost:443` (Server モード) を初期プロファイルとして生成・保存するロジックを追加。
- **管理者パスワードプロンプトダイアログ**:
  - パスワード認証失敗 (`Access has been denied`) 時に、パスワード入力用モーダルプロンプト (`promptConnectPassword`) を表示するダイアログフローを追加。
  - 入力されたパスワードはセッションメモリ上の `sessionPasswords` に保持され、以降の操作で非対話コマンドへ渡される。
- **初回接続時の管理者パスワード設定**:
  - 空のパスワードで接続が成功した初回接続時、Windows GUI版 サーバーマネージャーと同等に「初回接続: 新しい管理者パスワードを設定してください」というダイアログ (`promptInitialPassword`) を表示。
  - パスワードが入力された場合、`ServerPasswordSet` コマンドを発行して自動設定する。

### 11. パスワード誤入力時のフリーズ修正 & 並列コマンド実行による応答爆速化
- **パスワード誤入力時のハングアップ（signal: killed 待ち）の即時検出**:
  - `vpncmd` に誤ったパスワードを渡した際、`vpncmd` プロセスが対話的再入力プロンプトを表示して標準入力を待ち続け、タイムアウト（signal: killed）までフリーズしていた問題を修正。
  - `run` メソッド内で stdout/stderr に `Access has been denied` が含まれているかを即座に判定し、プロセス終了を待たずに直ちに `Access has been denied` エラーを返却してパスワード再入力モーダルを即時表示するよう改善。
- **接続初期化（`fetchServerInfo`）の並列 goroutine 化**:
  - `ServerInfo`, `ServerStatus`, `HubList` の3つの非対話 `vpncmd` 呼び出しを直列順次実行から goroutine による完全並列実行に変更。
  - 「Connecting...」の初期画面表示からダッシュボード描画までの待機時間を従来（直列実行）の約 1/3（最長1コマンドの所要時間のみ）へ大幅短縮・爆速化。

## 2026-11-03: M11 リモートアクセス設定 (IPsec/L2TP, OpenVPN, Dynamic DNS) の実装

### 1. リモートアクセス用 vpncmd コマンド群の統合
- `internal/vpncmd/client.go` に以下のメソッドおよびデータ構造を新設:
  - `IPsecGet` / `IPsecEnable`: L2TP/IPsec機能の有効化、事前共有鍵 (PSK)、デフォルトHubの設定
  - `OpenVpnGet` / `OpenVpnEnable` / `OpenVpnMakeConfig`: OpenVPN互換サーバーの有効化、UDPポート設定、クライアントconfig生成
  - `DynamicDnsGetStatus` / `DynamicDnsSetHostname`: DDNS状態取得（IPv4/IPv6、現在ホスト名）およびプレフィックスの変更
- `internal/vpncmd/client_test.go` にコマンド構築と引数の検証テストを追加。

### 2. TUI リモートアクセス画面 (`internal/ui/remoteaccess.go`) の新設
- ダッシュボードから `v` (VPN Remote Access) キーで遷移する新画面 (`screenRemoteAccess`) を実装。
- 画面構成:
  - **IPsec / L2TP / EtherIP 設定**: 有効/無効トグル、PSKテキスト入力、Default Hubテキスト入力
  - **OpenVPN 互換サーバー設定**: 有効/無効トグル、UDPポート番号テキスト入力
  - **ダイナミック DNS 状態**: DDNSホスト名編集、割り当て済みグローバルIPv4/IPv6の読み取り表示
- UI/UX 仕様への完全準拠:
  - 2段階状態遷移モデル（フィールド移動 ↔ テキスト入力モード）
  - テキスト入力中の `Esc` による画面遷移保護
  - 未保存変更 (`dirty`) 存在時の `Esc`/`q` 離脱防止ダイアログ (`confirmDiscardChanges`)
  - 未保存変更破棄専用キー `n` (`confirmDiscardInPlace`)
  - 保存（`s`）完了時のサーバーダッシュボード自動再取得 (`tea.Batch`)
- 多言語対応 (`internal/ui/catalog_en.go`) に全翻訳エントリを登録。
- 自動テスト (`app_ui_test.go`) に画面遷移・トグル動作・破棄保護のテストを追加。

## 2026-11-03: M12 Virtual HUB セキュリティ拡張 (接続元IP制限 Ac* & 信頼するCA証明書 CA*) の実装

### 1. 接続元IP制限および信頼するCA用 vpncmd コマンドの実装
- `internal/vpncmd/client.go` に以下のメソッドを追加:
  - `AcList` / `AcAdd` / `AcDel`: 送信元IPアドレスによるHUB接続許可/拒否ルールの一覧・追加・削除
  - `CAList` / `CAAdd` / `CADelete`: HUB単位で信頼するCA証明書の一覧・追加・削除
- `internal/vpncmd/client_test.go` に `AcAdd`, `CAAdd` の引数構築検証テストを追加。

### 2. Hub 詳細画面へのタブ拡張 ＆ 専用フォーム
- Hub 詳細画面に「接続元制限」(`hubTabAc`) および「信頼するCA」(`hubTabCA`) タブを新設。
- **接続元IP制限 (`hubTabAc`)**:
  - マルチカラム一覧テーブル表示、選択中ルールの詳細プロパティ表示。
  - 新規作成フォーム `acForm` (`screenAcForm` / `internal/ui/acform.go`): Action (Pass/Deny)、Priority、IP/サブネットの対話的入力フォーム。
  - 削除確認ダイアログ (`confirmDeleteAcRule`) による保護。
- **信頼するCA証明書 (`hubTabCA`)**:
  - CA証明書一覧テーブル表示。
  - `c` キーによる証明書ファイルパス入力モーダル (`promptAddCA`) を新設。
  - 削除確認ダイアログ (`confirmDeleteCA`) による保護。
- 多言語カタログ (`internal/ui/catalog_en.go`) に全27エントリを登録。
- 回帰テスト (`app_ui_test.go`) に各タブでの画面遷移、トグル、モーダル起動のテストを追加。

## 2026-11-04: マルチプラットフォーム配布拡張 (BSD各種 / macOS / Linux / Windows)

- **ターゲットプラットフォームの拡充 (計12環境)**:
  - Linux: `amd64`, `arm64`
  - macOS (Darwin): `arm64` (Apple Silicon), `amd64` (Intel)
  - FreeBSD: `amd64`, `arm64`
  - OpenBSD: `amd64`, `arm64`
  - NetBSD: `amd64`, `arm64`
  - Windows: `amd64`, `arm64`
- **設定ファイルの更新**:
  - `.goreleaser.yaml`: `freebsd`, `openbsd`, `netbsd` をビルド対象に追加し、全12ターゲットの tar.gz/zip アーカイブ生成に対応。
  - `install.sh`: `detect_os` に `FreeBSD`, `OpenBSD`, `NetBSD` を追加し、BSD環境でも `curl ... | sh` の自動インストールが動作するよう拡張。
  - `Makefile`: `PLATFORMS` に全12ターゲットを反映。`make cross` で手元一括ビルドが可能。
  - `.github/workflows/ci.yml`: 全12ターゲットのマトリクスビルド (`cross-build` ジョブ) を新設し、CI で常時クロスコンパイル互換性を検証。
- **ローカルビルド検証**: 全12ターゲットの Pure Go クロスコンパイルがエラーなく成功することを確認。



