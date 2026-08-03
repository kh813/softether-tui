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
- [x] `UserGet` による詳細閲覧および `UserSet` による属性（氏名・グループ・備考・パスワード・有効期限）のインライン編集・保存対応（専用 `UserDetail` 画面）

- [x] `UserCreate` (認証方式選択: Password / Anonymous / Radius に対応。NTLM/Certはパラメータ名未確認のため非対応)
- [x] `UserPasswordSet` パスワード再設定 (`p`キーで単独プロンプト)
- [x] `UserExpiresSet` 有効期限設定 (`e`キー、YYYY/MM/DD入力)
- [x] `UserDelete` (確認ダイアログ必須)
- [x] `GroupList` / `GroupCreate` / `GroupDelete` / `GroupGet` / `GroupSet` (専用 `GroupDetail` 画面で氏名・備考のインライン編集・保存対応)
- [x] ユーザーのグループ割当変更 (`g`キー、`UserSet /GROUP:`)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## M4: セッション監視 + ログ設定/閲覧 (重点機能)

- [x] `SessionList` 一覧表示 + 自動リフレッシュ (既定 5 秒、`+`/`-`キーで2〜60秒の範囲で変更可)
- [ ] `SessionGet` 詳細表示は見送り (一覧表示に統合。個別詳細画面は未実装)
- [x] `SessionDisconnect` (確認ダイアログ必須、`x`キー)
- [x] `LogGet` による閲覧に加え、`LogEnable`/`LogDisable`/`LogPacketSaveType`/`LogSwitchSet` によるログ設定変更機能
- [ ] ログファイル内容の閲覧 (tail 的ビュー、キーワードフィルタ) は未着手 (ローカル/リモートでのログ取得範囲の設計判断が必要。app_specs.md 12章の未決事項として残存)
- [x] ローカルで `go build` / `go vet` / `go test` 実行、バイナリの起動確認

## M5: リスナー管理、SecureNAT、アクセスリスト

- [x] `ListenerList` / `ListenerCreate` / `ListenerDelete` / `ListenerEnable` / `ListenerDisable` (ダッシュボードから`l`キーで独立画面。作成はポート番号プロンプト、有効/無効は`o`/`f`の明示操作)
- [x] `SecureNatEnable` / `SecureNatDisable` / `SecureNatStatusGet` / `SecureNatHostGet` はHub詳細のSecureNATタブで対応。
- [x] `SecureNatHostSet` (仮想ホストIP・MAC・サブネットマスクの編集) および `DhcpSet` (DHCP配布IP範囲・GW・DNS・ドメインの編集) の設定・変更機能（専用 `SecureNATDetail` 画面でインライン編集対応）
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

## M9: vpncmd_commands.md の公式マニュアル照合・網羅性チェック

「vpncmd の TUI ラッパーである以上、公式マニュアル記載のコマンドは基本的に全て実行可能にすべき」という方針の見直しを受け、`vpncmd_commands.md` を SoftEther 公式マニュアル (ja.softether.org 6.3〜6.6) と全面照合した。

- [x] 公式マニュアル 6.3 (サーバー全体)・6.4 (Virtual HUB)・6.5 (VPN Client)・6.6 (VPN Tools) を取得し、掲載されている全コマンドを抽出
- [x] `internal/vpncmd/client.go` の実コード (`Run`/`RunWithInput` に渡すコマンド文字列) を直接検索し、「実装済み」の自己申告と実態を突き合わせ
- [x] **誤って「実装済み `[x]`」になっていたが実際は未実装だった項目を修正**: `ServerCertGet`/`ServerCertSet`/`Caps`/`PolicyList`/`NatSet`
- [x] **コマンド名が誤っていた項目を修正**: `GetConfig`/`SetConfig`→`ConfigGet`/`ConfigSet`、`RebootServer`→`Reboot`、`Layer3*`→`Router*` (体系ごと異なる)、`ClusterSettingSet`→`ClusterSettingStandalone`/`Controller`/`Member`
- [x] **公式マニュアルに記載が見当たらないコマンドを「要確認」として明記**: `GetPerformance`、`LicenseList`系、`MakeCert2048`、`AccountDetailGet`
- [x] **丸ごと欠落していたコマンド群を追加 (90件超)**: 接続維持 (`Keep*`)、TCP接続管理 (`Connection*`)、IPsec/EtherIP/OpenVPN/SSTP/VPN Azure/DDNS、CA証明書管理 (`CA*`)、Cascade詳細設定 (`Cascade*` 約20コマンド)、拡張ACL (`AccessAddEx`/`Add6`/`AddEx6`)、接続元IP制限リスト (`Ac*`、`AccessList`とは別機能)、グループ参加/ポリシー、MAC/IPテーブル、NAT/DHCPテーブル、Hub拡張オプション、VPN Client Account詳細サブコマンド (約20コマンド) 等
- [x] `app_specs.md` 3.1/3.2 を更新: 「MVPスコープ」という限定的な表現から「vpncmd全コマンドの網羅を目標とする」方針に変更
- [ ] 今回追加した約90件の未実装コマンドの実装は今後のマイルストーンで順次対応 (優先度は `vpncmd_commands.md` の「対応方針」列 ✅/△/✕ を参照)
- [x] UI/UX の一貫性チェック (今回のユーザー指摘: 「他メンバーに実装してもらった際、見た目や操作の一貫性維持が難しかった」) — 実際に `internal/ui/*.go` を `app_specs.md` 8章のルールと突き合わせて調査した。結果は M10 に切り出した

## M10: UI/UX 一貫性の是正

M9 で実施した「`app_specs.md` 8章のUI/UXルールと実装の突き合わせ調査」で見つかった不整合を修正する。**経験の浅いプログラマーでも迷わず対応できるように、フェーズを細かく分割している。**

### 進め方のルール (全フェーズ共通)

- フェーズは **上から順番に** 進める (後のフェーズは前のフェーズの変更を前提にしている箇所がある)。
- 1つのフェーズが終わったら、必ず次の3点を確認してから次のフェーズに進む:
  1. `go build ./...` がエラーなく通ること
  2. `go vet ./...` がエラーなく通ること
  3. `go test ./...` が全て PASS すること (`internal/ui` の `TestEnCatalogCoversAllSourceStrings` は新しい日本語文言を追加したら英語カタログにも追加しないと失敗するので注意)
- 上記3点を確認できたら、そのフェーズの変更だけを1つの `git commit` にする (フェーズをまとめて1コミットにしない)。コミットメッセージは各フェーズの見出しをそのまま使ってよい。
- わからないことがあれば、自己判断で仕様を変えずに、コメントで質問を残すか一旦中断してよい。

### 背景 (なぜ直すのか)

現在、`c` キーが画面によって意味が違う。
- 一覧画面 (`app.go` のダッシュボード類, `dashboard.go`, `bridge.go`, `clientdashboard.go`, `listener.go`, Hub詳細のユーザー/グループタブ) では `c` = **新規作成 (Create)**。
- 一部の「フィールド編集パネル」画面 (`userdetail.go`, `groupdetail.go`, `securenatdetail.go`, Hub詳細の SecureNAT タブ) では `c` = **編集内容の破棄 (Cancel)**。

これは `app_specs.md` §8.1.1 の必須ルール5「Create は常に `c`、Cancel/No は常に `n`」に反しており、ユーザーが画面をまたいで操作するときに混乱する直接の原因になっている。さらに、この4画面では `c` での破棄には確認ダイアログが出ない一方、`Esc` での破棄には確認ダイアログが出るという、同じ「破棄」操作なのに安全性が異なるバグもある。

### Phase 1: ヘルプバーの見た目統一 (`accountform.go` / `bridgeform.go`)

一番簡単で影響範囲が狭いフェーズ。他の全画面はキー操作ヘルプを共通関数 `renderHelp()` (`internal/ui/styles.go`) で描画しており、キー名が色付きハイライトされる。この2ファイルだけ素の文字列を `dimStyle.Render(...)` で描画しており、キーがハイライトされない。

- [x] `internal/ui/accountform.go` の `View()` 内、以下の行を探す:
  ```go
  b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  ←→: 認証方式切替  Enter: 作成  Esc: キャンセル")))
  ```
  これを `renderHelp()` を使った形に書き換える:
  ```go
  b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "←→", tr("認証方式切替"), "Enter", tr("作成"), "Esc", tr("キャンセル")))
  ```
- [x] `internal/ui/bridgeform.go` の `View()` 内、以下の行を探す:
  ```go
  b.WriteString("\n" + dimStyle.Render(tr("Tab/↑↓: 項目移動  ←→: TAP切替  Enter: 作成  Esc: キャンセル")))
  ```
  同様に書き換える:
  ```go
  b.WriteString("\n" + renderHelp("Tab/↑↓", tr("項目移動"), "←→", tr("TAP切替"), "Enter", tr("作成"), "Esc", tr("キャンセル")))
  ```
- [x] `internal/ui/app_ui_test.go` に、この2画面のヘルプ行が `renderHelp` 経由になったことを確認する軽いテストを追加する (既存の他画面向けテストを参考に、`View()` の出力に ANSI カラーコードが含まれること、または既存の `renderHelp` を使ったテストパターンを流用してよい)。
  - 実装メモ: `go test` はttyなしで実行されるため lipgloss は ANSI コードを出力しない。そのため「色が付くこと」ではなく、`renderHelp` が `"key"+":"+"desc"` (コロン前にスペースなし) で連結するのに対し旧実装は `"key: desc"` (コロン前にスペースあり) だった、という文字列上の違いで判定する `TestAccountFormAndBridgeFormUseSharedHelpRenderer` を追加した。
- [x] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 1: accountform/bridgeform のヘルプバーを renderHelp() に統一")

### Phase 2: 「その場で破棄」用の共通確認ダイアログを追加する (基盤整備)

Phase 3〜6 で使う共通の仕組みをここで先に作る。**ここが一番概念的に難しいフェーズなので、慎重に進める。**

現在、`Esc` で破棄する場合は必ず「本当に破棄しますか?」の確認ダイアログ (`confirmDiscardChanges`) が出るが、これは確認後に **前の画面に戻る** 動作までセットになっている (`internal/ui/app.go` の `confirmDiscardChanges` ケース、`m.screen = screenHubDetail` の行を参照)。Phase 3〜6 で直すショートカットは「画面には留まったまま、入力内容だけ破棄する」動作なので、画面遷移をしない新しい確認種別 `confirmDiscardInPlace` を追加する。

- [x] `internal/ui/confirm.go` の `confirmKind` の `const` 一覧に、`confirmDiscardChanges` の次の行として `confirmDiscardInPlace` を追加する。
- [x] `internal/ui/app.go` で `confirmDiscardChanges` を処理している `case confirmDiscardChanges:` ブロック (だいたい1410行目付近) を探す。その直後に、以下のような新しい `case` を追加する (`m.screen = screenHubDetail` の代入を **入れない** ことがポイント。画面を移動させず、その場に留まる):
  ```go
  case confirmDiscardInPlace:
      if m.screen == screenUserDetail {
          m.userDetail.editedValues = make(map[editableUserField]string)
          m.userDetail.dirty = false
          m.userDetail.authType = vpncmd.UserAuthNone
      } else if m.screen == screenGroupDetail {
          m.groupDetail.editedValues = make(map[editableGroupField]string)
          m.groupDetail.pendingMemberEdits = make(map[string]bool)
          m.groupDetail.dirty = false
      } else if m.screen == screenSecureNATDetail {
          m.secureNatDetail.editedValues = make(map[editableSecureNATField]string)
          m.secureNatDetail.dirty = false
      } else if m.screen == screenHubDetail && m.hubDetail.tab == hubTabSecureNAT {
          m.hubDetail.secureNatEditedValues = make(map[editableSecureNATField]string)
          m.hubDetail.secureNatDirty = false
      }
      m.status = tr("変更を破棄しました")
      m.statusErr = false
      return m, nil
  ```
  (`screenSecureNATDetail` 用の分岐は、調査の過程で見つかった別の不整合の修正でもある: 現行コードでは `screenSecureNATDetail` 画面で `Esc` 経由の破棄確認をしても、既存の `confirmDiscardChanges` ケースにはこの画面用の分岐がなく `dirty` フラグと編集内容がリセットされていなかった。)
- [x] このフェーズだけでは呼び出し元がまだないので、動作確認は「コンパイルが通ること」のみでよい (Phase 3〜6 で実際に呼び出す)。
- [x] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 2: その場に留まって破棄する confirmDiscardInPlace を追加")

### Phase 3: `securenatdetail.go` の `c` キーを `n` に変更する

`internal/ui/securenatdetail.go` の `Update()` 内、以下のブロックを探す:
```go
case "c", "C":
    if d.dirty {
        d.editedValues = make(map[editableSecureNATField]string)
        d.dirty = false
        m.status = tr("変更を破棄しました")
        m.statusErr = false
        return m, nil
    }
```

- [x] 上記ブロックを、キーを `n`/`N` に変え、直接破棄するのではなく Phase 2 で作った確認ダイアログを呼ぶ形に書き換える:
  ```go
  case "n", "N":
      if d.dirty {
          m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
          return m, nil
      }
  ```
- [x] `internal/ui/catalog_en.go` に `"未保存の変更があります。変更を破棄しますか?"` の英訳エントリを追加する (既存の似た文言 `"未保存の変更があります。変更を破棄して戻りますか?"` の訳を参考にする)。
- [x] `View()` 内のヘルプ表示行 (`renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更"), "s", tr("保存 (Save)"), "c", tr("変更を破棄 (Cancel)"))`) の `"c"` を `"n"` に変更する。
- [x] `internal/ui/app_ui_test.go` に「SecureNAT詳細画面で dirty な状態で `n` を押すと確認ダイアログが出て、`y` で確定すると `dirty` が `false` に戻り画面は同じ (`screenSecureNATDetail`) のままであること」を確認するテストを追加する (`TestSecureNATDetailDiscardKeyIsNNotC`。合わせて `c` キーを押しても何も起きないことも確認)。
- [x] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 3: securenatdetail.go の破棄キーを c から n に変更")

### Phase 4: `userdetail.go` の `c` キーを `n` に変更する

Phase 3 と全く同じパターン。`internal/ui/userdetail.go` の `Update()` 内、以下のブロックを探す:
```go
case "c", "C":
    if d.dirty {
        d.editedValues = make(map[editableUserField]string)
        d.dirty = false
        d.authType = vpncmd.UserAuthNone
        m.status = tr("変更を破棄しました")
        m.statusErr = false
        return m, nil
    }
```

- [x] キーを `n`/`N` に変え、確認ダイアログ経由にする:
  ```go
  case "n", "N":
      if d.dirty {
          m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
          return m, nil
      }
  ```
- [x] `View()` 内のヘルプ表示行の `"c"` を `"n"` に変更する。
- [x] `internal/ui/app_ui_test.go` に Phase 3 と同様のテストをユーザー詳細画面向けに追加する (`TestUserDetailDiscardKeyIsNNotC`)。
- [x] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 4: userdetail.go の破棄キーを c から n に変更")

### Phase 5: `groupdetail.go` の `c` キーを整理する (注意: この画面だけ `c` に2つの役割がある)

`groupdetail.go` は他の3画面と違い、`c`/`C` に **2つの役割** が同居している (`a`/`A` と同じ「グループメンバー追加」、かつ dirty時は「破棄」)。**「メンバー追加」の役割はそのまま残し、「破棄」の役割だけを `n` に移す** のがこのフェーズの目的。

- [x] `internal/ui/groupdetail.go` の `Update()` 内、以下のブロックを探す:
  ```go
  case "c", "C":
      if d.dirty {
          d.editedValues = make(map[editableGroupField]string)
          d.pendingMemberEdits = make(map[string]bool)
          d.dirty = false
          m.status = tr("変更を破棄しました")
          m.statusErr = false
          return m, nil
      }
      m.prompt.Show(promptAddGroupMember, d.groupName, fmt.Sprintf(tr("グループ %q に追加するユーザー名:"), d.groupName), tr("ユーザー名"), false)
      return m, nil
  ```
  これを、dirty時は何もしない (メンバー追加を許可しない、という既存の挙動は維持) ように単純化し、新しく `n`/`N` のケースを追加する:
  ```go
  case "c", "C":
      if !d.dirty {
          m.prompt.Show(promptAddGroupMember, d.groupName, fmt.Sprintf(tr("グループ %q に追加するユーザー名:"), d.groupName), tr("ユーザー名"), false)
          return m, nil
      }

  case "n", "N":
      if d.dirty {
          m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
          return m, nil
      }
  ```
- [x] 同じファイル内、画面上の `[ キャンセル (Cancel) ]` ボタン (カーソルを合わせて `Enter`/`Space` で実行するボタン。`"cancelText"` 変数のあたり) を Enter で実行している箇所も、直接破棄せず同じ確認ダイアログを経由するように変更する。該当箇所 (`case " ", "enter":` 内、`d.cursor == 3+len(users)` で保存、それ以外で破棄している部分) を探し、破棄側の分岐を次のように変更する:
  ```go
  m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
  return m, nil
  ```
  (直接 `d.editedValues = ...` 等をリセットしていた行は削除する。カーソル位置のリセット (`d.cursor = 0`) も確認ダイアログの結果を待ってから行うべきだが、今回は挙動を変えすぎないよう、リセットは省略してよい。気になる場合はコメントで相談する。)
- [x] `View()` 内のヘルプ表示行の `"c"` (破棄の意味で使っている方) を `"n"` に変更する。「メンバー追加」の意味で `"c"`/`"a"` を表示している箇所はそのまま残す。
- [x] `internal/ui/app_ui_test.go` に「グループ詳細画面で dirty でない状態で `c` を押すとメンバー追加プロンプトが開くこと」「dirty な状態で `n` を押すと確認ダイアログが開くこと」の2つを確認するテストを追加する (既存の `TestGroupDetailAddMemberPrompt` を拡張)。
- [x] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 5: groupdetail.go の破棄キーを c から n に変更 (メンバー追加の c は維持)")

### Phase 6: Hub詳細の SecureNAT タブ (`app.go` 側) の `c` キーを `n` に変更する

これは `internal/ui/hubdetail.go` の View ではなく、キー処理は `internal/ui/app.go` の Hub詳細画面の `Update` 処理内にある。

- [ ] `internal/ui/app.go` 内、以下のブロックを探す (`hubTabSecureNAT` のキー処理の中):
  ```go
  case "c", "C":
      if d.secureNatDirty {
          d.secureNatEditedValues = make(map[editableSecureNATField]string)
          d.secureNatDirty = false
          m.status = tr("変更を破棄しました")
          m.statusErr = false
  ```
  (この続きの行も含めて) キーを `n`/`N` に変え、確認ダイアログ経由にする:
  ```go
  case "n", "N":
      if d.secureNatDirty {
          m.confirm.Show(confirmDiscardInPlace, "", tr("未保存の変更があります。変更を破棄しますか?"))
          return m, nil
      }
  ```
- [ ] `internal/ui/hubdetail.go` の `viewSecureNAT()` 内、ヘルプ表示行 (`renderHelp("↑/↓", tr("項目選択"), "Enter", tr("値の変更/切替"), "s", tr("保存 (Save)"), "c", tr("変更を破棄 (Cancel)"))`) の `"c"` を `"n"` に変更する。
- [ ] `internal/ui/app_ui_test.go` に Phase 3 と同様のテストを Hub詳細画面の SecureNAT タブ向けに追加する。
- [ ] `go build ./...` / `go vet ./...` / `go test ./...` を実行しエラーがないことを確認
- [ ] `git commit` (例: "Phase 6: Hub詳細 SecureNAT タブの破棄キーを c から n に変更")

### Phase 7: 最終確認・仕上げ

- [ ] リポジトリ全体で `grep -rn '"c", "C"' internal/ui/*.go` を実行し、破棄 (Cancel/Discard) の意味で `c` が使われている箇所が残っていないことを目で確認する (Create/新規作成/メンバー追加の意味で使われている `c` は残ってよい)。
- [ ] `internal/ui/app_ui_test.go` を通し実行し、Phase 1・3〜6 で追加したテストを含め全て PASS することを確認する。
- [ ] `vpncmd_commands.md` は今回の変更と無関係なので変更不要 (念のため確認のみ)。
- [ ] `app_specs.md` に変更履歴として1行追記する (例: 8章の末尾または改訂履歴節に「2026-08: `c`/`n` キーの衝突を解消 (Create=c, Cancel/Discard=n に統一)」)。
- [ ] 本ファイル (`app_todo.md`) の M10 の各チェックボックスが全て `[x]` になっていることを確認する。
- [ ] 最終確認として `go build ./...` / `go vet ./...` / `go test ./...` / `golangci-lint run` を実行し、全て問題ないことを確認する。
- [ ] `git commit` (例: "Phase 7: M10 UI/UX 一貫性是正の仕上げ・ドキュメント更新")

### このマイルストーンで対応しないこと (別タスク・将来対応)

- `groupdetail.go` にのみ存在する画面上の `[ 保存 ]`/`[ キャンセル ]` ボタン行を、`userdetail.go`/`securenatdetail.go` にも同様に追加するかどうかの検討 (レイアウト・カーソル数の変更を伴う大きめの変更のため、今回のキーバインド統一とは別タスクとする)。
- `accountform.go`/`bridgeform.go` のような「常時入力編集」形式のフォームと、`userdetail.go` 等の「フィールド選択→Enterで編集」形式のフォームが混在している点。両方とも用途に応じた合理的な設計であり、現時点では不整合とはみなさないが、将来的に `app_specs.md` 8章の「2フェーズ状態モデル」の適用範囲を明文化する際に再検討する。

## 将来 (11章 12章 未決事項)

- [ ] VPN Client の証明書認証・NIC管理・信頼するCA管理・自動接続設定・設定インポート/エクスポート (M7で見送った項目、M9で公式コマンド名を確認済み)
- [ ] VPN Tools モード (`/TOOLS`) 対応 (M9で対象コマンドを確認済み)
- [ ] JSON-RPC 管理 API 方式の検討
- [ ] OS キーチェーン連携 (パスワード保存)
- [ ] リリース成果物の署名 (cosign 等)
- [ ] 英語/日本語以外の言語への対応
- [ ] groupdetail.go 以外の詳細編集画面への画面上 Save/Cancel ボタン行の追加検討 (M10 で対応を見送った項目)
