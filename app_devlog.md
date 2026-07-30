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

### 4. モーダルレイアウト枠外溢れ修正 & Virtual HUB 詳細状態コマンド修正
- **モーダル表示時のレイアウト枠外溢れ修正**:
  - `prompt` や `confirm` ダイアログアクティブ時に、親画面テキストに縦追加（改行連結）されていたためターミナル枠線外にはみ出す問題を修正。
  - モーダル有効時は全画面コンテンツをモーダル自身に差し替えて1つの枠線内に収めて描画するレイアウト構造に改修。
- **Virtual HUB 詳細取得コマンドの修正 (`StatusGet`)**:
  - SoftEther vpncmd には `GetHub` や `HubGet` コマンドが存在せず `exit status 117` となる現象が発生していたため、対象 Hub コンテキスト (`/HUB:<name>`) で実行可能な正式コマンド **`StatusGet`** に修正。
  - `vpncmd_commands.md` および `internal/vpncmd/client.go` を更新。
