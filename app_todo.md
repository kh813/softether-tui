# 実装 ToDo リスト

- 本ファイルは `app_specs.md` の 10 章 (マイルストーン) に沿った実装タスクの進捗管理用。
- 完了したタスクは `go build` / `go test` 等で動作確認した上で `[x]` にする。
- vpncmd コマンド単位の対応状況は `vpncmd_commands.md` 側のチェックボックスも合わせて更新する。

## M0: プロジェクト基盤

- [x] Go module 初期化 (`go.mod`)
- [x] ディレクトリ構成 (`internal/config`, `internal/vpncmd`, `internal/ui`)
- [x] 依存ライブラリ導入 (bubbletea, bubbles, lipgloss, yaml)
- [x] `main.go` エントリポイント
- [x] ローカルで `go build` が通ることを確認

## M1: 接続プロファイル管理 + 接続テスト + サーバーダッシュボード

- [x] プロファイルデータモデル定義 (`Profile` 構造体: 名前・ホスト・ポート・モード Server/Bridge・Hub・パスワード保存先)
- [x] プロファイル永続化 (`~/.config/softether-tui/profiles.yaml`、XDG準拠)
- [x] プロファイル CRUD のユニットテスト
- [x] vpncmd 実行アダプタ: バイナリ探索 (`exec.LookPath`)、非対話コマンド実行 (`/CMD:...`)
- [x] vpncmd アダプタ: `ServerInfoGet` / `ServerStatusGet` 呼び出し・CSV パース
- [x] vpncmd アダプタのユニットテスト (実バイナリ不要、CSV 出力のパース処理のみ検証)
- [x] TUI: プロファイル選択画面 (一覧・追加・編集・削除フォーム)
- [x] TUI: 削除確認ダイアログ (共通コンポーネント)
- [x] TUI: 接続テスト実行・結果表示
- [x] TUI: サーバーダッシュボード画面 (ServerInfoGet/ServerStatusGet 表示。Hub一覧は M2)
- [x] 画面遷移: プロファイル選択 → (接続テスト/接続) → ダッシュボード、`Esc` で戻る
- [x] ローカルで `go build` + バイナリ起動確認 (vpncmd 未インストール環境でもクラッシュしないこと)

## M2: Hub 一覧・作成・削除・基本設定

- [x] `HubList` 取得・一覧表示 (ダッシュボードから遷移)
- [x] `HubGet` による Hub 詳細閲覧 (概要タブ)。`HubSet` によるフィールド単位の編集は未着手 (パラメータ名が要確認のため見送り。オンライン/オフロインは専用コマンドで対応 = 次項)
- [x] `HubCreate` / `HubDelete` (削除は確認ダイアログ必須)
- [x] Hub オンライン/オフライン切替 (Hub 管理モードの `Online` / `Offline` コマンド。トグルではなく `o`=オンライン化 / `f`=オフライン化 の明示操作)
- [x] Hub 詳細タブの土台 (概要 / ユーザー / グループ / セッション / セキュリティログ / SecureNAT / ACL / カスケード) を切替可能にする。概要タブ以外は「今後のマイルストーンで実装予定」のプレースホルダー
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認 (擬似端末上でクラッシュしないことを確認)

## M3: ユーザー / グループ管理 (重点機能)

- [x] `UserList` 一覧表示・検索フィルタ (`/`で検索文字列入力→Enterで確定、名前以外の列も部分一致対象)
- [x] `UserGet` による詳細閲覧は見送り、一覧表示に統合。`UserSet` は次項のグループ変更のみ対応 (他フィールド編集はパラメータ名要確認のため未着手)
- [ ] `UserSet` によるユーザーの氏名 (`/REALNAME:`)・備考 (`/NOTE:`) の編集機能 (`e`キーで編集フォーム呼び出し)
- [x] `UserCreate` (認証方式選択: Password / Anonymous / Radius に対応。NTLM/Certはパラメータ名未確認のため非対応)
- [x] `UserPasswordSet` パスワード再設定 (`p`キーで単独プロンプト)
- [x] `UserExpiresSet` 有効期限設定 (`e`キー、YYYY/MM/DD入力)
- [x] `UserDelete` (確認ダイアログ必須)
- [x] `GroupList` / `GroupCreate` / `GroupDelete`。`GroupSet`(表示名/備考の編集)は未着手
- [x] ユーザーのグループ割当変更 (`g`キー、`UserSet /GROUP:`)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## M4: セッション監視 + ログ設定/閲覧 (重点機能)

- [x] `SessionList` 一覧表示 + 自動リフレッシュ (既定 5 秒、`+`/`-`キーで2〜60秒の範囲で変更可)
- [ ] `SessionGet` 詳細表示は見送り (一覧表示に統合。個別詳細画面は未実装)
- [x] `SessionDisconnect` (確認ダイアログ必須、`x`キー)
- [x] `LogGet` による閲覧のみ対応。`LogEnable`/`LogPacketSaveType`/`LogSwitchType`によるログ設定変更は未着手 (vpncmdの正確な引数構文が未確認のため意図的に見送り。5章/12章の既存の未決事項と同じ理由)
- [ ] ログファイル内容の閲覧 (tail 的ビュー、キーワードフィルタ) は未着手 (ローカル/リモートでのログ取得範囲の設計判断が必要。app_specs.md 12章の未決事項として残存)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## M5: リスナー管理、SecureNAT、アクセスリスト

- [x] `ListenerList` / `ListenerCreate` / `ListenerDelete` / `ListenerEnable` / `ListenerDisable` (ダッシュボードから`l`キーで独立画面。作成はポート番号プロンプト、有効/無効は`o`/`f`の明示操作)
- [x] `SecureNatEnable` / `SecureNatDisable` / `SecureNatStatusGet` / `SecureNatHostGet` はHub詳細のSecureNATタブで対応。
- [ ] `SecureNatHostSet` (仮想ホストIP・MAC・サブネットマスクの編集: `s`キー) および `DhcpSet` (DHCP配布IP範囲・GW・DNSの編集: `h`キー) の設定・変更機能
- [x] `AccessList` / `AccessDelete` / `AccessEnable` / `AccessDisable` はHub詳細のACLタブで対応。`AccessAdd`(ルール追加)は未着手 (優先度/送受信IP/ポート範囲/プロトコル等の引数構文が未確認のため意図的に見送り)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## M6: カスケード接続・ローカルブリッジ管理 (Bridge モード)

- [x] `CascadeList` / `CascadeDelete` はHub詳細の「カスケード」タブで対応。`CascadeCreate`(新規接続の作成)は未着手 (リモートホスト/ポート/Hub/認証方式の引数構文が未確認のため意図的に見送り)
- [x] `CascadeStatusGet` / `CascadeDetailGet` はクライアント実装のみ (UI側は一覧表示に統合し個別詳細画面は見送り。他コマンドと同様の判断)
- [x] `CascadeOnline` / `CascadeOffline` (`o`/`f`キー)
- [x] `BridgeList` / `BridgeCreate` / `BridgeDelete` (ローカルブリッジ)。ダッシュボードから`b`キーで独立画面。作成時は`BridgeDeviceList`で利用可能デバイスを参照表示
- [x] `/BRIDGE` モードでの接続プロファイル動作確認 — 単体テストで `Target.Mode=ModeBridge` の場合に全コマンドが `/SERVER` ではなく `/BRIDGE` フラグを使うことを確認。実際の VPN Bridge サーバーでの動作確認は未実施 (本プロジェクト全体で実サーバー未接続のため)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## CI/CD・配布 (11章)

- [x] PR/main 向け GitHub Actions (`.github/workflows/ci.yml`: `go build` / `go vet` / `go test` / `golangci-lint`)
- [x] `.goreleaser.yaml` (クロスコンパイルターゲット: linux/darwin/windows × amd64/arm64、`goreleaser check` と `--snapshot --clean` でのビルドを実機検証済み)
- [x] タグ push トリガーのリリースワークフロー (`.github/workflows/release.yml`、goreleaser-action)
- [x] `checksums.txt` 生成 (`.goreleaser.yaml` の `checksum` セクション、スナップショットビルドで生成を確認)
- [x] `install.sh` (curl ワンライナーインストール。ローカルHTTPサーバーでダウンロード→チェックサム検証→展開→`~/.local/bin`配置→バージョン確認まで実地検証済み)
- [x] `Makefile` (ローカルでの(クロス)コンパイル用。`build`/`run`/`install`/`fmt`/`vet`/`test`/`lint`/`check`/`cross`/`checksums`/`clean`/`help`の各ターゲットを実装し、`make check`・`make build VERSION=...`・`make cross`(5プラットフォーム全て)・`make checksums`を実行して動作確認済み)
- [x] `main.go`に`commit`/`date`変数を追加。`.goreleaser.yaml`が注入する`main.commit`/`main.date`のldflagsが実は変数未定義で黙って無視されていたバグを修正し、`--version`出力にコミットハッシュとビルド日時を追加表示するようにした
- [x] リポジトリパスを実値 `kh813/softether-tui` に差し替え (`.goreleaser.yaml` の `release.github.owner`、`install.sh` の `REPO` と使用例コメント)
- [x] 初回push後のCI失敗を修正: `ci.yml` の `golangci-lint-action@v6` (v1系CLI前提) が `version: latest` で導入されたv2系golangci-lintと非互換 (v6が渡す `--out-format` フラグをv2 CLIが未サポート、exit code 3) だったため `golangci-lint-action@v7` に更新。ローカルで同事象を再現の上、修正後 `golangci-lint run` が0件で通ることを確認

## M7: VPN Client モード対応 (`/CLIENT`、前倒し着手)

当初は「将来」ロードマップだったが、ユーザー要望により前倒しで着手した。

- [x] `buildArgs` に `/CLIENT` モードを追加。Client モードでは `/HUB:` を付与しないよう分岐
- [x] `config.ModeClient` を追加し、プロファイル編集フォームでServer/Bridge/Clientの3モードを循環選択できるようにした
- [x] `AccountList` 一覧表示 (Hubの概念がないため専用の「VPN Client管理」画面をプロファイル選択から直接遷移させる形で新設。既存のサーバーダッシュボード/Hub詳細とは別画面)
- [x] `AccountCreate` (接続名・サーバーホスト:ポート・接続先Hub・認証方式[Password/Anonymous]を指定するフォーム)
- [x] `AccountDelete` (確認ダイアログ必須)
- [x] `AccountConnect` / `AccountDisconnect` (`o`/`f`キー、他画面と同じ明示操作の規則)
- [x] `AccountPasswordSet` (`p`キーでプロンプト、既存のUser向けパスワード再設定と同じ仕組みを再利用)
- [x] 接続テスト (`t`キー) をモードに応じて分岐: Server/BridgeはServerInfoGet、ClientはAccountListを使用
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認
- [ ] `AccountCertSet`(証明書認証)、`NicList`/`NicCreate`/`NicDelete`(NIC管理)、`TrustCAList`等(信頼するCA管理)、`AccountStartupSet`/`Remove`(自動接続)、`AccountExport`/`Import`(設定のインポート/エクスポート)は意図的に未対応 (パラメータ構文未確認、または優先度が低いため)

## M8: 多言語対応 (i18n、文字化け対応のため前倒し着手)

日本語ロケールのないサーバーで日本語UI文字列・罫線記号が文字化けし実用に耐えないとの指摘を受け、前倒しで着手した (6.5参照)。

- [x] `internal/i18n` パッケージ: `Lang`型、`Detect()`(LC_ALL→LC_MESSAGES→LANGの順でロケール判定、非日本語/未設定は英語にフォールバック)、`Parse()`(`--lang`値のバリデーション)
- [x] `main.go`: `--lang en`/`--lang ja` フラグを追加し、未指定時は`i18n.Detect()`の結果を使用
- [x] `internal/ui`: 既存の日本語決め打ち文字列を翻訳カタログ経由 (`tr()`関数、`enCatalog`マップ) に置き換え。カタログは日本語文字列をキーとした英語訳のマップ方式 (フィールド名を新規に考案する構造体方式は変更範囲が大きすぎるため不採用)。関数名は当初`t()`としたが、テストコードの`t *testing.T`慣習と衝突するため`tr()`に変更
- [x] 罫線 (`borderStyle`) ・状態記号 (`●`/`✕`) を英語モード時はアスキーセーフな表現 (`+`/`-`/`|`罫線、`[OK]`/`[FAIL]`) に切り替え。矢印 (`↑↓←→`) は英訳文言側で "Up/Down"/"Left/Right" 等に置き換える形で対応 (記号自体の別モードは用意していない)
- [x] `internal/i18n` のユニットテスト (`Detect`のロケール優先順位・判定、`Parse`のバリデーション)
- [x] `internal/ui`のユニットテスト: 全ソースの`tr("...")`呼び出しをASTで静的走査し`enCatalog`に対応翻訳が存在することを検証するテスト (`TestEnCatalogCoversAllSourceStrings`)。実装中に見つかった不足エントリはこのテストで検出・修正済み
- [x] ローカルで `go build` / `go vet` / `go test` 実行 (全パッケージPASS)。`--lang en`/`--lang ja`起動時のCLIメッセージ(vpncmd未検出警告等)が言語別に正しく出力されることを非対話実行で確認。擬似端末上でのTUI描画の目視確認は今回の環境では安定して再現できず未実施 (自動テストとCLI出力確認でカバー)
- [x] vpncmd自体の出力(エラーメッセージ等)は翻訳対象外である旨は`app_specs.md` 6.5節に明記済み (README未作成のため6.5節が一次情報源)

## 将来 (11章 12章 未決事項)

- [ ] VPN Client の証明書認証・NIC管理・信頼するCA管理・自動接続設定・設定インポート/エクスポート (M7で見送った項目)
- [ ] VPN Tools モード (`/TOOLS`) 対応
- [ ] JSON-RPC 管理 API 方式の検討
- [ ] OS キーチェーン連携 (パスワード保存)
- [ ] リリース成果物の署名 (cosign 等)
- [ ] 英語/日本語以外の言語への対応
