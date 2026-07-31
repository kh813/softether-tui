# vpncmd コマンド一覧・実装状況チェックリスト

- 目的: `vpncmd` が提供するコマンドを一覧化し、本 TUI ラッパー (`softether-tui`) でどこまで対応しているかを管理する。
- 位置づけ: `app_specs.md` は設計意図・機能要件を記述する仕様書、本ファイルは実装カバレッジを追跡する生きたチェックリストとして役割を分離する。
- 免責事項: 以下はコマンド名・分類について実装知識をもとに作成した初版ドラフト。SoftEther のバージョンにより存在しないコマンドや名称差異がありうるため、実装着手前に対象バージョンの `vpncmd` (`/CMD:Help`、対話モードでの `?`) や公式マニュアルで正式名称・引数を確認すること。「要確認」と付記した項目は特に注意。

## 凡例

- **MVP対象**
  - ✅: 本仕様書 (`app_specs.md`) 5 章の機能要件に含まれる、MVP で対応する操作
  - △: Server/Bridge モードのコマンドだが 5 章の機能要件には未記載 (Enterprise 版限定機能など)。将来検討
  - —: VPN Client / VPN Tools モードのコマンド。`app_specs.md` 3.2 のとおり将来ロードマップ
- **実装状況**
  - `[ ]` 未着手 / `[~]` 一部対応 (一覧表示のみ等) / `[x]` 実装済み

---

## 1. VPN Server 管理コマンド (Server Admin Mode)

`/SERVER` で接続し、Hub を選択していない状態 (サーバー全体・Hub横断の操作) で実行できるコマンド群。

### 1.1 基本情報・接続

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| About | バージョン・ビルド情報の表示 | ✅ | [x] | `ServerInfoGet` を利用してダッシュボード画面上部に表示 |
| Caps | サーバーの対応機能一覧の取得 | ✅ | [x] | クライアント `Caps` 実装完了。内部フラグ取得として利用 |
| Check | 動作環境チェック (TUN/TAP 等) | △ | [ ] | サーバー接続前・OS診断用ユーティリティのためTUI管理画面対象外 |
| Flush | 設定をディスクへ書き込み | △ | [ ] | vpncmd が設定変更時に自動的にディスク保存するため手動実行不要 |
| GetPerformance | パフォーマンス統計の取得 | △ | [ ] | リアルタイムモニターは将来ロードマップ (M8) にて検討 |
| Quit / Exit | vpncmd の終了 (TUI では不要) | — | [ ] | TUI アプリ自体の終了 (`q` キー) で代替するため不要 |

### 1.2 サーバー基本設定

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| ServerInfoGet | サーバー情報 (バージョン/OS等) 取得 | ✅ | [x] | ダッシュボード上部に自動表示 |
| ServerStatusGet | サーバー状態 (セッション数等) 取得 | ✅ | [x] | ダッシュボード上部に自動表示 |
| ServerPasswordSet | サーバー管理パスワード設定 | △ | [x] | 初回接続時ダイアログおよび設定画面にて対応 |
| ServerCertGet / ServerCertSet | サーバー証明書取得/設定 | ✅ | [ ] | X.509証明書ファイルパス指定インターフェースは将来拡張機能 |
| ServerCipherGet / ServerCipherSet | 暗号化アルゴリズム取得/設定 | △ | [x] | クライアント実装完了 |
| VpnOverIcmpDnsGet / VpnOverIcmpDnsEnable | ICMP/DNS 経由 VPN の状態取得/有効・無効化 | △ | [x] | クライアント実装完了 |
| SysLogEnable / SyslogDisable / SysLogGet | syslog 転送設定の取得/有効・無効化 | △ | [x] | クライアント実装完了 |
| GetConfig / SetConfig | サーバー設定のテキストエクスポート/インポート | △ | [ ] | 設定ファイルのテキスト直接編集・書き戻しはCLI/ファイル操作を推奨 |
| RebootServer | VPN Server サービスの再起動 | △ | [ ] | 意図しない切断を防止するためTUI直操作からは除外 |

### 1.3 リスナー管理

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| ListenerList | リスナー一覧取得 | ✅ | [x] | リスナー画面 `l` で表示 |
| ListenerCreate | リスナー作成 | ✅ | [x] | リスナー画面 `a` キーで作成 |
| ListenerDelete | リスナー削除 | ✅ | [x] | リスナー画面 `d` キーで削除 |
| ListenerEnable | リスナー有効化 | ✅ | [x] | リスナー画面 `o` キーで有効化 |
| ListenerDisable | リスナー無効化 | ✅ | [x] | リスナー画面 `f` キーで無効化 |

### 1.4 Virtual HUB 管理 (作成・削除等はサーバー側から)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| HubList | Hub 一覧取得 | ✅ | [x] | ダッシュボードに一覧表示 |
| HubCreate | Hub 作成 (スタンドアロン) | ✅ | [x] | ダッシュボード `a` キーで作成 |
| HubCreateDynamic / HubCreateStatic | 動的/静的 Hub 作成 | △ | [ ] | クラスタ環境向け機能。スタンドアロン Hub 作成で代替 |
| HubDelete | Hub 削除 | ✅ | [x] | ダッシュボード `d` キーで削除 |
| StatusGet | Hub 詳細状態取得 | ✅ | [x] | Hub 概要タブで全自動表示 |
| SetHubPassword | Hub パスワード設定 | ✅ | [x] | ダッシュボード `p` キーで設定プロンプト起動 |
| Hub | 指定 Hub を選択し Hub 管理モードへ遷移 | ✅ | [x] | TUI タブ切替時に `/HUB:<name>` オプションでシームレス遷移 |

### 1.5 ローカルブリッジ・Layer3 スイッチ

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| BridgeDeviceList | ブリッジ可能な物理 NIC/tap 一覧 | ✅ | [x] | ブリッジ画面で自動参照 |
| BridgeList | ローカルブリッジ一覧 | ✅ | [x] | ブリッジ画面 `b` で一覧表示 |
| BridgeCreate | ローカルブリッジ作成 | ✅ | [x] | ブリッジ画面 `a` キーで作成 |
| BridgeDelete | ローカルブリッジ削除 | ✅ | [x] | ブリッジ画面 `d` キーで削除 |
| Layer3List / Layer3AddIf / Layer3DelIf / Layer3AddRoute / Layer3DelRoute / Layer3Enable / Layer3Disable | 仮想 Layer3 スイッチ管理 | △ | [ ] | 複合L3ルーティング設定は別画面コンポーネントとして将来設計予定 |

### 1.6 ライセンス・クラスタ管理 (Enterprise/Cluster Edition 向け)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| LicenseList / LicenseAdd / LicenseDel / LicenseStatusGet | ライセンスキー管理 | △ | [ ] | Enterprise版専用機能 |
| ClusterSettingGet / Set / ClusterMemberList / ClusterConnectionStatusGet | クラスタ構築・構成管理 | △ | [ ] | オープンソース版SoftEtherの基本運用範囲外 |

---

## 2. Virtual HUB 管理コマンド (Hub Management Mode)

`Hub <名前>` で対象 Hub を選択した後に実行できるコマンド群。

### 2.1 Hub 基本操作

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| Online | 選択中 Hub をオンライン化 | ✅ | [x] | Hub画面 `o` キー |
| Offline | 選択中 Hub をオフライン化 | ✅ | [x] | Hub画面 `f` キー |
| OptionsGet | Hub 動作オプション取得 | ✅ | [x] | Hub概要画面で自動表示 |
| SetMaxSession | 最大接続セッション数設定 | ✅ | [x] | クライアント実装完了 |
| SetEnumAllow / SetEnumDeny | 匿名ユーザーによるHub列挙許可/拒否 | ✅ | [x] | クライアント実装完了 |

### 2.2 ユーザー管理 (重点機能)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| UserList | ユーザー一覧取得 | ✅ | [x] | Usersタブで自動取得・検索対応 |
| UserCreate | ユーザー作成 | ✅ | [x] | Usersタブ `a` キーで作成 |
| UserGet | ユーザー詳細取得 | ✅ | [x] | ユーザー詳細画面でインライン編集対応 |
| UserSet | ユーザー詳細変更 | ✅ | [x] | グループ・氏名・備考などのインライン編集 |
| UserDelete | ユーザー削除 | ✅ | [x] | Usersタブ `d` キーで削除 |
| UserPasswordSet | パスワード認証への設定/再設定 | ✅ | [x] | ユーザー詳細画面で `p` キー再設定 |
| UserAnonymousSet | 匿名認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserRadiusSet | RADIUS 認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserNTLMSet | NTLM (Windowsドメイン) 認証への変更 | △ | [ ] | ADドメイン参加環境固有のため未対応 |
| UserCertSet / UserSignedSet | X.509証明書/署名付き証明書認証 | △ | [ ] | クライアント証明書ファイル指定プロンプト未実装 |
| PolicyList | セキュリティポリシー定義一覧の取得 | ✅ | [x] | UserGetおよびクライアントで全38項目自動参照 |
| UserPolicySet | ユーザーポリシー (帯域制限・接続制限) 設定 | ✅ | [x] | クライアント実装完了 |
| UserPolicyRemove | ユーザーポリシー削除 | ✅ | [x] | クライアント実装完了 |
| UserExpiresSet | アカウント有効期限設定 | ✅ | [x] | ユーザー詳細画面でインライン変更 |

### 2.3 グループ管理

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| GroupList | グループ一覧取得 | ✅ | [x] | Groupsタブで一覧表示 |
| GroupCreate | グループ作成 | ✅ | [x] | Groupsタブ `a` キーで作成 |
| GroupGet | グループ詳細取得 | ✅ | [x] | グループ詳細画面で表示・編集 |
| GroupSet | グループ詳細変更 | ✅ | [x] | 氏名・備考のインライン編集対応 |
| GroupDelete | グループ削除 | ✅ | [x] | Groupsタブ `d` キーで削除 |

### 2.4 セッション管理 (重点機能)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| SessionList | 現在の接続セッション一覧 | ✅ | [x] | Sessionsタブでリアルタイム自動更新 |
| SessionGet | セッション詳細取得 | ✅ | [x] | 一覧テーブルの各行データとして統合表示 |
| SessionDisconnect | セッション強制切断 | ✅ | [x] | Sessionsタブ `x` キーで切断 |

### 2.5 SecureNAT / DHCP / RADIUS

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| SecureNatEnable / SecureNatDisable | SecureNAT 全体有効化/無効化 | ✅ | [x] | SecureNATタブ `o/f` キー |
| SecureNatStatusGet | SecureNAT 状態取得 | ✅ | [x] | SecureNATタブで表示 |
| SecureNatHostGet / SecureNatHostSet | 仮想ホスト (IP/MAC/Subnet) 設定取得/変更 | ✅ | [x] | SecureNATタブでインライン編集 |
| NatEnable / NatDisable | Virtual NAT 個別有効化/無効化 | ✅ | [x] | SecureNATタブ `n/N` キー |
| NatGet / NatSet | NAT 動作設定取得/変更 | ✅ | [x] | NatGet取得対応。設定はSecureNatHostSetに統合 |
| DhcpEnable / DhcpDisable | Virtual DHCP 個別有効化/無効化 | ✅ | [x] | SecureNATタブ `h/H` キー |
| DhcpGet / DhcpSet | DHCP 配布範囲・リース時間設定取得/変更 | ✅ | [x] | SecureNATタブでインライン編集 |
| RadiusServerGet / RadiusServerSet / RadiusServerDelete | RADIUS サーバー設定取得/変更/削除 | ✅ | [x] | Hub概要 `R` キーでモーダル設定 |

### 2.6 アクセスリスト (パケットフィルタ)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AccessList | アクセスリスト一覧取得 | ✅ | [x] | ACLタブで表示 |
| AccessAdd | アクセスリストルール追加 | ✅ | [x] | ACLタブ `a` キーでプロンプト追加 |
| AccessDelete | アクセスリストルール削除 | ✅ | [x] | ACLタブ `d` キーで削除 |
| AccessEnable / AccessDisable | ルール有効化/無効化 | ✅ | [x] | ACLタブ `o/f` キーで切替 |

### 2.7 カスケード接続 (拠点間接続)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CascadeList | カスケード接続一覧 | ✅ | [x] | Cascadeタブで表示 |
| CascadeCreate | カスケード接続作成 | ✅ | [x] | Cascadeタブ `a` キーで作成 |
| CascadeGet / CascadeSet | カスケード接続詳細設定取得/変更 | ✅ | [x] | 作成プロンプトおよび一覧で管理 |
| CascadeDelete | カスケード接続削除 | ✅ | [x] | Cascadeタブ `d` キーで削除 |
| CascadeStatusGet / CascadeDetailGet | カスケード接続状態・詳細取得 | ✅ | [x] | クライアント実装完了 |
| CascadeOnline / CascadeOffline | オンライン/オフライン化切替 | ✅ | [x] | Cascadeタブ `o/f` キーで切替 |

### 2.8 セキュリティログ / パケットログ

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| LogGet | ログ設定取得 | ✅ | [x] | Logタブで表示 |
| LogEnable / Disable | セキュリティ/パケットログ有効化/無効化 | ✅ | [x] | Logタブで Enter/Space 切替 |
| LogPacketSaveType | パケットログ保存種別設定 (DHCP/TCP/UDP等) | ✅ | [x] | Logタブで Enter/Space 切替 |
| LogSwitchSet | ログファイル切り替え周期設定 | ✅ | [x] | Logタブで Enter/Space 切替 |
| LogFileList / LogFileGet | ログファイル一覧・リモートダウンロード | △ | [ ] | TUI内での大容量ログ閲覧は負荷が高いため、ローカルログファイル参照を推奨 |

### 2.9 証明書失効リスト (CRL)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CrlList / CrlAdd / CrlDel | 証明書失効リスト (CRL) 管理 | △ | [ ] | PKI・電子証明書運用環境専用 |

---

## 3. VPN Bridge 管理コマンド (`/BRIDGE` モード)

VPN Bridge は VPN Server のサブセット実装であり、ローカルブリッジおよびカスケード接続操作は Server モードと共通のTUIUIで操作可能。

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| BridgeList / BridgeCreate / BridgeDelete | ローカルブリッジ一覧・作成・削除 | ✅ | [x] | Serverモードと共通画面で操作可能 |
| CascadeList / CascadeCreate / CascadeDelete / CascadeOnline / CascadeOffline | カスケード接続一覧・作成・切断等 | ✅ | [x] | Serverモードと共通画面で操作可能 |

---

## 4. VPN Client 管理コマンド (`/CLIENT` モード)

| コマンド | 概要 | MVP対象 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AccountList | 接続設定 (アカウント) 一覧 | ✅ | [x] | Client画面で表示 |
| AccountCreate | 接続設定作成 | ✅ | [x] | Client画面 `a` キーで作成 |
| AccountGet / AccountDetailGet | 接続設定詳細・統計取得 | ✅ | [x] | クライアント `AccountGet` / `AccountDetailGet` 実装完了 |
| AccountSet | 接続設定変更 | ✅ | [x] | ユーザー名 `u` キー、パスワード `p` キーで変更対応 |
| AccountDelete | 接続設定削除 | ✅ | [x] | Client画面 `d` キーで削除 |
| AccountConnect / AccountDisconnect | 接続開始/切断 | ✅ | [x] | Client画面 `o` / `f` キーで操作 |
| AccountStatusGet | 接続状態・統計取得 | ✅ | [x] | クライアント `AccountStatusGet` 実装完了 |
| AccountUsernameSet | 接続ユーザー名変更 | ✅ | [x] | Client画面 `u` キーでプロンプト変更対応 |
| AccountPasswordSet | パスワード認証設定/変更 | ✅ | [x] | Client画面 `p` キーでプロンプト変更対応 |
| AccountAnonymousSet | 匿名認証への変更 | ✅ | [x] | クライアント実装完了 |
| AccountCertSet | 証明書認証への変更 | △ | [ ] | クライアントPKI証明書指定プロンプト未実装 |
| NicList / NicCreate / NicDelete | 仮想 NIC 一覧・作成・削除 | ✅ | [x] | クライアント `NicList` / `NicCreate` / `NicDelete` バックエンド実装完了 |

---

## 5. VPN Tools コマンド (`/TOOLS` モード)

> `app_specs.md` 3.2 のとおり MVP 非対象。サーバー/クライアント接続を伴わないユーティリティ群。実装時は要確認。

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| MakeCert | 証明書・秘密鍵の生成 | — | [ ] |
| Check | 動作環境チェック (TUN/TAP・依存ライブラリ等) | — | [ ] |
| TrafficClient | 通信性能測定 (クライアント側、要確認) | — | [ ] |
| TrafficServer | 通信性能測定 (サーバー側、要確認) | — | [ ] |
| PortChecker | ポート到達性チェック (要確認、コマンド名不確か) | — | [ ] |

---

## 更新方針

- 実装を進めるたびに「実装状況」列を更新する。
- 「要確認」の項目は実装着手時に正式なコマンド名・パラメータを確認し、確定したら注記を削除する。
- 存在しないことが判明したコマンドは打ち消し線を引くか行を削除し、新たに判明したコマンドは追記する。
