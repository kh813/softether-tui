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

### 2. vpncmd コマンド名の不一致修正
- **`GetHub` コマンドへの修正**:
  - `HubGet` コマンドが存在せず `exit status 117` となっていた箇所を、/HUB: コンテキストで動作する正式なコマンド `GetHub` に変更。
  - `vpncmd_commands.md` のチェックリストも合わせて最新化。

### 3. バグ修正と単体テスト・i18n追従
- 同一ディレクトリおよびカレントディレクトリからの `vpncmd` 自動探索 (`Locate`) をサポート。
- 新規追加した UI メッセージ 6 件を `internal/ui/catalog_en.go` に英訳追加し、`TestEnCatalogCoversAllSourceStrings` の自動テストを PASS。
- 全パッケージ (`internal/config`, `internal/i18n`, `internal/ui`, `internal/vpncmd`) の `go test ./...` が正常通過することを確認。
