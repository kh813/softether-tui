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

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| About | バージョン・ビルド情報の表示 | ✅ | [ ] |
| Caps | サーバーの対応機能一覧の取得 | ✅ | [ ] |
| Check | 動作環境チェック (TUN/TAP 等) | △ | [ ] |
| Flush | 設定をディスクへ書き込み | △ | [ ] |
| GetPerformance | パフォーマンス統計の取得 | △ | [ ] |
| Quit / Exit | vpncmd の終了 (TUI では不要) | — | [ ] |

### 1.2 サーバー基本設定

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| ServerInfoGet | サーバー情報 (バージョン/OS等) 取得 | ✅ | [x] |
| ServerStatusGet | サーバー状態 (セッション数等) 取得 | ✅ | [x] |
| ServerPasswordSet | サーバー管理パスワード設定 | △ | [x] |
| ServerCertGet | サーバー証明書取得 | ✅ | [ ] |
| ServerCertSet | サーバー証明書設定 | ✅ | [ ] |
| ServerCipherSet | 使用暗号化アルゴリズム設定 (要確認) | △ | [ ] |
| VpnOverIcmpDnsGet | ICMP/DNS 経由 VPN の有効状態取得 (要確認) | △ | [ ] |
| VpnOverIcmpDnsEnable / Disable | ICMP/DNS 経由 VPN の有効/無効化 (要確認) | △ | [ ] |
| SysLogEnable / Disable | syslog 転送設定 | △ | [ ] |
| SysLogGet | syslog 転送設定の取得 | △ | [ ] |
| GetConfig | サーバー設定全体をテキストでバックアップ (要確認) | △ | [ ] |
| SetConfig | サーバー設定全体を復元 (要確認) | △ | [ ] |
| RebootServer | VPN Server サービスの再起動 | △ | [ ] |

### 1.3 リスナー管理

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| ListenerList | リスナー一覧取得 | ✅ | [x] |
| ListenerCreate | リスナー作成 | ✅ | [x] |
| ListenerDelete | リスナー削除 | ✅ | [x] |
| ListenerEnable | リスナー有効化 | ✅ | [x] |
| ListenerDisable | リスナー無効化 | ✅ | [x] |

### 1.4 Virtual HUB 管理 (作成・削除等はサーバー側から)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| HubList | Hub 一覧取得 | ✅ | [x] |
| HubCreate | Hub 作成 (スタンドアロン) | ✅ | [x] |
| HubCreateDynamic | Hub 作成 (動的種別、要確認) | △ | [ ] |
| HubCreateStatic | Hub 作成 (静的種別、要確認) | △ | [ ] |
| HubDelete | Hub 削除 | ✅ | [x] |
| StatusGet | Hub 詳細状態取得 | ✅ | [x] |
| SetHubPassword | Hub パスワード設定 | ✅ | [ ] |
| Hub | 指定 Hub を選択し Hub 管理モードへ遷移 | ✅ | [~] (対話選択ではなく `/HUB:` 接続オプションで同等の効果を実現) |

### 1.5 ローカルブリッジ・Layer3 スイッチ

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| BridgeDeviceList | ブリッジ可能な物理 NIC/tap 一覧 | ✅ | [x] |
| BridgeList | ローカルブリッジ一覧 | ✅ | [x] |
| BridgeCreate | ローカルブリッジ作成 | ✅ | [x] |
| BridgeDelete | ローカルブリッジ削除 | ✅ | [x] |
| Layer3List | 仮想 Layer3 スイッチ一覧 (要確認) | △ | [ ] |
| Layer3AddIf / Layer3DelIf | Layer3 スイッチのインターフェース追加/削除 (要確認) | △ | [ ] |
| Layer3AddRoute / Layer3DelRoute | Layer3 スイッチのルート追加/削除 (要確認) | △ | [ ] |
| Layer3Enable / Layer3Disable | Layer3 スイッチの有効/無効化 (要確認) | △ | [ ] |

### 1.6 ライセンス・クラスタ管理 (Enterprise/Cluster Edition 向け、要確認)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| LicenseList | ライセンス一覧 | △ | [ ] |
| LicenseAdd | ライセンス追加 | △ | [ ] |
| LicenseDel | ライセンス削除 | △ | [ ] |
| LicenseStatusGet | ライセンス状態取得 | △ | [ ] |
| ClusterSettingGet / Set | クラスタ設定取得/変更 | △ | [ ] |
| ClusterMemberList | クラスタメンバー一覧 | △ | [ ] |
| ClusterConnectionStatusGet | クラスタ接続状態取得 | △ | [ ] |

---

## 2. Virtual HUB 管理コマンド (Hub Management Mode)

`Hub <名前>` で対象 Hub を選択した後に実行できるコマンド群。

### 2.1 Hub 基本操作

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| Online | 選択中 Hub をオンライン化 | ✅ | [x] |
| Offline | 選択中 Hub をオフライン化 | ✅ | [x] |
| OptionsGet | Hub 動作オプション取得 (要確認) | ✅ | [ ] |
| OptionsSet | Hub 動作オプション変更 (要確認) | ✅ | [ ] |

### 2.2 ユーザー管理 (重点機能)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| UserList | ユーザー一覧取得 | ✅ | [x] |
| UserCreate | ユーザー作成 | ✅ | [x] |
| UserGet | ユーザー詳細取得 | ✅ | [x] (専用のユーザー詳細画面でインライン編集対応) |
| UserSet | ユーザー詳細変更 | ✅ | [x] (グループ、氏名、備考などの編集に対応) |
| UserDelete | ユーザー削除 | ✅ | [x] |
| UserPasswordSet | パスワード認証への設定/再設定 | ✅ | [x] |
| UserAnonymousSet | 匿名認証への変更 | ✅ | [x] |
| UserRadiusSet | RADIUS 認証への変更 | ✅ | [x] |
| UserNTLMSet | NTLM 認証への変更 (要確認) | ✅ | [ ] (意図的に未対応。パラメータ名要確認) |
| UserCertSet | 証明書認証への変更 (要確認、コマンド名要確認) | ✅ | [ ] (意図的に未対応。パラメータ名要確認) |
| UserPolicySet | ユーザーポリシー (帯域制限等) 設定 | ✅ | [ ] |
| UserPolicyRemove | ユーザーポリシー削除 | ✅ | [ ] |
| UserExpiresSet | アカウント有効期限設定 | ✅ | [x] |

### 2.3 グループ管理

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| GroupList | グループ一覧取得 | ✅ | [x] |
| GroupCreate | グループ作成 | ✅ | [x] |
| GroupGet | グループ詳細取得 | ✅ | [x] (専用のグループ詳細画面でインライン編集対応) |
| GroupSet | グループ詳細変更 (所属ユーザーの割当含む、要確認) | ✅ | [x] (氏名、備考の編集に対応) |
| GroupDelete | グループ削除 | ✅ | [x] |

### 2.4 セッション管理 (重点機能)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| SessionList | 現在の接続セッション一覧 | ✅ | [x] |
| SessionGet | セッション詳細取得 | ✅ | [ ] (一覧表示に統合し個別詳細画面は未実装) |
| SessionDisconnect | セッション強制切断 | ✅ | [x] |

### 2.5 SecureNAT / DHCP / RADIUS

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| SecureNatEnable | SecureNAT 有効化 | ✅ | [x] |
| SecureNatDisable | SecureNAT 無効化 | ✅ | [x] |
| SecureNatStatusGet | SecureNAT 状態取得 | ✅ | [x] |
| SecureNatHostGet | SecureNAT (仮想ホスト) 設定取得 | ✅ | [x] |
| SecureNatHostSet | SecureNAT (仮想ホスト) 設定変更 | ✅ | [x] |
| NatGet | NAT 設定取得 | ✅ | [ ] |
| NatSet | NAT 設定変更 | ✅ | [ ] |
| DhcpGet | DHCP 配布設定取得 | ✅ | [x] |
| DhcpSet | DHCP 配布設定変更 | ✅ | [x] |
| RadiusServerGet | RADIUS サーバー設定取得 | ✅ | [x] |
| RadiusServerSet | RADIUS サーバー設定変更 | ✅ | [x] |
| RadiusServerDelete | RADIUS サーバー設定削除 | ✅ | [x] |

### 2.6 アクセスリスト (パケットフィルタ)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| AccessList | アクセスリスト一覧取得 | ✅ | [x] |
| AccessAdd | アクセスリストルール追加 | ✅ | [ ] (意図的に未対応。優先度/IP/ポート範囲/プロトコル等の引数構文が未確認) |
| AccessDelete | アクセスリストルール削除 | ✅ | [x] |
| AccessEnable | アクセスリストルール有効化 | ✅ | [x] |
| AccessDisable | アクセスリストルール無効化 | ✅ | [x] |

### 2.7 カスケード接続 (拠点間接続、Bridge 用途含む)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| CascadeList | カスケード接続一覧 | ✅ | [x] |
| CascadeCreate | カスケード接続作成 | ✅ | [ ] (意図的に未対応。リモートホスト/ポート/Hub/認証方式の引数構文が未確認) |
| CascadeGet | カスケード接続設定取得 (要確認) | ✅ | [ ] |
| CascadeSet | カスケード接続設定変更 (要確認) | ✅ | [ ] |
| CascadeDelete | カスケード接続削除 | ✅ | [x] |
| CascadeStatusGet | カスケード接続状態取得 | ✅ | [~] (クライアント実装のみ。UI画面は未接続) |
| CascadeDetailGet | カスケード接続詳細取得 | ✅ | [~] (クライアント実装のみ。UI画面は未接続) |
| CascadeOnline | カスケード接続をオンライン化 | ✅ | [x] |
| CascadeOffline | カスケード接続をオフライン化 | ✅ | [x] |

### 2.8 セキュリティログ / パケットログ (重点機能)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| LogGet | ログ設定取得 | ✅ | [x] |
| LogEnable / Disable | ログ (セキュリティ/パケット) 有効化/無効化 | ✅ | [x] |
| LogPacketSaveType | パケットログの保存種別設定 | ✅ | [x] |
| LogSwitchType | ログファイルの切り替え周期設定 | ✅ | [x] |
| LogFileList | ログファイル一覧 (要確認) | ✅ | [ ] (ログ取得範囲の設計判断が未確定。app_specs.md 12章参照) |
| LogFileGet | ログファイル内容取得 (要確認) | ✅ | [ ] (同上) |

### 2.9 証明書失効リスト (CRL)

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| CrlList | CRL 登録一覧 | △ | [ ] |
| CrlAdd | CRL 追加 | △ | [ ] |
| CrlDel | CRL 削除 | △ | [ ] |

### 2.10 リモートアクセス機能拡張 (IPsec/L2TP・OpenVPN・SSTP・DDNS、要確認)

サーバー全体設定 / Hub 単位設定のどちらに属するかはバージョンにより異なる可能性があるため実装時に要確認。

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| IPsecEnable | IPsec/L2TP 事前共有鍵等の設定 (要確認) | △ | [ ] |
| IPsecGet | IPsec/L2TP 設定取得 (要確認) | △ | [ ] |
| OpenVpnMakeConfig | OpenVPN クライアント設定ファイル生成 | △ | [ ] |
| DDnsGet | DDNS 設定・現在のホスト名取得 | △ | [ ] |
| DDnsSet | DDNS ホスト名設定 (要確認) | △ | [ ] |
| DDnsStatusGet | DDNS 状態取得 (要確認) | △ | [ ] |

---

## 3. VPN Bridge 管理コマンド (`/BRIDGE` モード)

VPN Bridge は VPN Server のサブセット的な実装であり、コマンド体系の多くは 1 章・2 章と共通 (`ServerPasswordSet`, `ServerCertGet/Set`, `ListenerList/Create/Delete`, `BridgeList/Create/Delete`, `CascadeList/Create/Delete/Online/Offline` 等)。VPN Bridge 固有・中心となる操作は以下の通り。

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| BridgeList | ローカルブリッジ一覧 | ✅ | [x] (Server/Bridge共通実装。実Bridgeサーバーでの動作は未検証) |
| BridgeCreate | ローカルブリッジ作成 | ✅ | [x] (同上) |
| BridgeDelete | ローカルブリッジ削除 | ✅ | [x] (同上) |
| CascadeList | カスケード (拠点間) 接続一覧 | ✅ | [x] (同上) |
| CascadeCreate | カスケード接続作成 | ✅ | [ ] (2章と同じ理由で未対応) |
| CascadeDelete | カスケード接続削除 | ✅ | [x] (同上) |
| CascadeOnline / Offline | カスケード接続のオンライン/オフライン切替 | ✅ | [x] (同上) |
| ListenerList / Create / Delete | リスナー管理 (1.3 と共通) | ✅ | [x] (同上) |
| ServerCertGet / Set | 証明書管理 (1.2 と共通) | ✅ | [ ] |

---

## 4. VPN Client 管理コマンド (`/CLIENT` モード)

> `app_specs.md` 3.2 の当初ロードマップでは将来項目だったが、ユーザー要望により M7 として前倒しで着手した (5.10 参照)。NIC管理・信頼するCA管理・証明書認証・自動接続設定・設定インポート/エクスポートは非スコープのまま。

| コマンド | 概要 | MVP対象 | 実装状況 |
|---|---|---|---|
| AccountList | 接続設定 (アカウント) 一覧 | ✅ | [x] |
| AccountCreate | 接続設定作成 | ✅ | [x] |
| AccountSet | 接続設定変更 | ✅ | [ ] (フィールド単位の編集は未着手。UserSet/GroupSetと同様の判断) |
| AccountDelete | 接続設定削除 | ✅ | [x] |
| AccountConnect | 接続開始 | ✅ | [x] |
| AccountDisconnect | 接続切断 | ✅ | [x] |
| AccountStatusGet | 接続状態取得 (要確認) | ✅ | [~] (クライアント実装のみ。一覧表示に統合しUI画面は未接続) |
| AccountUsernameSet | 接続ユーザー名変更 | ✅ | [~] (クライアント実装のみ。UI画面には未接続) |
| AccountPasswordSet | 接続パスワード変更 | ✅ | [x] |
| AccountAnonymousSet | 匿名認証への変更 | ✅ | [x] |
| AccountCertSet | 証明書認証への変更 (要確認) | △ | [ ] (意図的に未対応。パラメータ名未確認) |
| AccountRetrySet | 再接続設定 | △ | [ ] |
| AccountStartupSet / AccountStartupRemove | スタートアップ接続設定 (要確認、OS依存) | △ | [ ] |
| AccountExport / AccountImport | 接続設定のエクスポート/インポート | △ | [ ] |
| NicList | 仮想 NIC 一覧 | △ | [ ] |
| NicCreate | 仮想 NIC 作成 | △ | [ ] |
| NicDelete | 仮想 NIC 削除 | △ | [ ] |
| NicUpgrade | 仮想 NIC アップグレード (要確認) | △ | [ ] |
| TrustCAList | 信頼する CA 証明書一覧 (要確認) | △ | [ ] |
| TrustCAAdd / TrustCADelete | 信頼する CA 証明書の追加/削除 (要確認) | △ | [ ] |

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
