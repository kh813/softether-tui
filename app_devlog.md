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
