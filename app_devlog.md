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

### 7. 接続タイムアウト最適化 & CSVテーブル解析ロジックの抜本的改修
- **タイムアウト時間の短縮 (15s -> 5s)**:
  - `vpncmd.Client` のデフォルトタイムアウトを 15秒から 5秒へ短縮し、「Connecting...」表示の待ち時間を大幅に削減。
- **`ParseCSVTable` による `Password`/`Item`/`Value` メタデータ行・列の自動除去**:
  - `vpncmd` の出力に含まれる「Password:」「Item,Value」などの副次的メタデータ行・ヘッダー行を `ParseCSVTable` で解析段階から除去。
  - 実在する Virtual Hub のデータ行のみが確実に取得・表示され、上位の `DEFAULT` Hub がデフォルトカーソル位置で選択されるよう改修。
