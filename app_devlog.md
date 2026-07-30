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

### 10. vpncmd サブメニュー機能の実装 (UserSet, SecureNatHostSet, DhcpSet)
- **仕様書 (`app_specs.md`) および ToDo リスト (`app_todo.md`) の更新**:
  - `app_specs.md` (5.5, 5.6) および `app_todo.md` (M3, M5) に `UserSet` (氏名・備考の編集), `SecureNatHostSet` (仮想ホスト IP 設定), `DhcpSet` (DHCP 範囲設定) の機能仕様とタスクを追加更新。
- **`vpncmd` アダプタ (`internal/vpncmd/client.go`) のメソッド拡張**:
  - `UserSet`: `/REALNAME:`, `/NOTE:` オプションを受け取る構造体とコマンド送信ロジックを追加。
  - `SecureNatHostSet`: `/IP:`, `/MASK:`, `/MAC:` オプションを受け取る構造体とコマンド送信ロジックを追加。
  - `DhcpGet` / `DhcpSet`: `/START:`, `/END:`, `/MASK:`, `/GW:`, `/DNS:` 等の DHCP オプションを受け取る構造体とコマンド送信ロジックを追加。
- **UI 操作キーバインド・入力プロンプトの連動**:
  - SecureNAT タブにて `i` キーで仮想ホスト IP アドレス設定 (`promptSecureNatHostIP`)、`s` キーで DHCP 範囲設定 (`promptDhcpStart`) のプロンプトダイアログを起動・実行できるよう拡張。
