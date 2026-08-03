# vpncmd コマンド一覧・実装状況チェックリスト

- 目的: `vpncmd` が提供するコマンドを一覧化し、本 TUI ラッパー (`softether-tui`) でどこまで対応しているかを管理する。
- 位置づけ: `app_specs.md` は設計意図・機能要件を記述する仕様書、本ファイルは実装カバレッジを追跡する生きたチェックリストとして役割を分離する。
- **本アプリの方針**: 「vpncmd の TUI ラッパー」であるため、公式マニュアルに掲載されている vpncmd コマンドは基本的に全て TUI から実行可能にすることを目標とする (Enterprise 限定機能・サポート専用コマンド・廃止済みコマンドなど、実質的に対応不可能なものを除く)。
- **典拠**: 本チェックリストは SoftEther 公式マニュアル (日本語版) Chapter 6 "コマンドラインリファレンスマニュアル" を典拠として作成・検証した。
  - [6.1 vpncmd について](https://ja.softether.org/4-docs/1-manual/6/6.1)
  - [6.2 vpncmd の一般的な使い方](https://ja.softether.org/4-docs/1-manual/6/6.2)
  - [6.3 VPN Server/VPN Bridge 管理コマンド (サーバー全体)](https://ja.softether.org/4-docs/1-manual/6/6.3)
  - [6.4 VPN Server/VPN Bridge 管理コマンド (Virtual HUB)](https://ja.softether.org/4-docs/1-manual/6/6.4)
  - [6.5 VPN Client 管理コマンド](https://ja.softether.org/4-docs/1-manual/6/6.5)
  - [6.6 VPN Tools コマンド](https://ja.softether.org/4-docs/1-manual/6/6.6)
- **実装状況の検証方法**: 「実装状況」列は `internal/vpncmd/client.go` 内で実際に vpncmd へ送信しているコマンド文字列 (`Run`/`RunWithInput` の引数) を直接検索して確認したものであり、コメントや変数名からの推測ではない。過去のドラフトには「実装済み」と誤記されていたが実際にはコードが存在しないコマンドが複数あったため、本更新で全項目を再検証し修正した (下記「2026-08 改訂」参照)。
- 免責事項: コマンド名・引数は公式マニュアルの記述を基にしているが、SoftEther のバージョンにより増減・変更がありうる。「要確認」と付記した項目は実装着手時に対象バージョンの `vpncmd` (`/CMD:Help`、対話モードでの `?`) でも再確認すること。

## 2026-08 改訂の概要

前回ドラフトを公式マニュアル 6.3〜6.6 と照合し、以下を修正した:

1. **コマンド名の誤り修正**:
   - `GetConfig`/`SetConfig` → `ConfigGet`/`ConfigSet` (語順が逆だった)
   - `RebootServer` → `Reboot`
   - `Layer3List`/`Layer3AddIf`/`Layer3DelIf`/`Layer3AddRoute`/`Layer3DelRoute`/`Layer3Enable`/`Layer3Disable` → `RouterList`/`RouterAdd`/`RouterDelete`/`RouterStart`/`RouterStop`/`RouterIfList`/`RouterIfAdd`/`RouterIfDel`/`RouterTableList`/`RouterTableAdd`/`RouterTableDel` (コマンド体系自体が異なる)
   - `ClusterSettingSet` (想定) → `ClusterSettingStandalone`/`ClusterSettingController`/`ClusterSettingMember` の3コマンドに分かれている
2. **「実装済み」の誤記を修正 (実際は未実装)**: `ServerCertGet`/`ServerCertSet`/`Caps`/`PolicyList`/`NatSet` は `client.go` にコードが存在せず、誤って `[x]` になっていたため `[ ]` に修正した。
3. **公式マニュアルに記載が見当たらないコマンド**: `GetPerformance`、`LicenseList`/`LicenseAdd`/`LicenseDel`/`LicenseStatusGet`、`MakeCert2048`、`AccountDetailGet` (`AccountDetailSet` のみ確認できた) は現行マニュアルに掲載がなく、存在しないか旧バージョン限定の可能性がある。「要確認」として保持。
4. **丸ごと欠落していたコマンド群を追加** (合計 90 件超): 接続維持 (`Keep*`)、TCP接続管理 (`Connection*`)、IPsec/EtherIP/OpenVPN/SSTP/VPN Azure/DDNS 関連、CA証明書管理 (`CA*`)、Cascade詳細設定 (`Cascade*` 約20コマンド)、拡張ACL (`AccessAddEx`/`AccessAdd6`/`AccessAddEx6`)、接続元IP制限リスト (`Ac*`、`AccessList` とは別機能)、グループ参加/ポリシー (`GroupJoin`/`GroupUnjoin`/`GroupPolicy*`)、MAC/IPテーブル、NAT/DHCPテーブル、Hub拡張オプション (`AdminOption*`/`ExtOption*`)、VPN Client の Account 詳細サブコマンド (約20コマンド)、証明書/スマートカード管理、Remote管理、Keep-Alive。

## 凡例

- **対応方針**
  - ✅: 通常運用で使われうる操作であり、TUI での対応を目指す
  - △: Enterprise/クラスタライセンス、AD/NTLMドメイン、スマートカード等の特殊環境に依存する、またはレガシー/低頻度機能のため優先度は低いが対応は検討する
  - ✕: TUI での実装が実質的に不可能・無意味 (サポート専用コマンド、廃止済みコマンド等)
- **実装状況**
  - `[ ]` 未着手 / `[~]` 一部対応 (一覧表示のみ・バックエンドのみ等) / `[x]` 実装済み (`client.go` でのコマンド送信を確認済み)

---

## 1. VPN Server / VPN Bridge 管理コマンド (サーバー全体、6.3)

`/SERVER` または `/BRIDGE` で接続し、Hub を選択していない状態で実行できるコマンド群。

### 1.1 基本情報・バージョン

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| About | vpncmd 自体のバージョン情報表示 | ✅ | [~] | TUI 自身のバージョン (`--version`) を表示。vpncmd バイナリ側のバージョンではない点に注意 |
| ServerInfoGet | サーバー情報 (バージョン/OS等) 取得 | ✅ | [x] | ダッシュボード上部に自動表示 |
| ServerStatusGet | サーバー状態 (セッション数等) 取得 | ✅ | [x] | ダッシュボード上部に自動表示 |
| Caps | サーバーの対応機能一覧の取得 | △ | [ ] | 未実装 (旧ドラフトで `[x]` と誤記されていたが `client.go` に該当コードなし) |
| Check | 動作環境チェック (TUN/TAP 等) | ✕ | [ ] | サーバー接続前の OS 診断用ユーティリティのため TUI 管理画面の対象外 (5章 Tools 参照) |
| GetPerformance | 公式マニュアルに記載が見当たらない | — | [ ] | 要確認: 現行マニュアルに掲載なし。存在しないコマンドの可能性 |

### 1.2 サーバー基本設定・証明書

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| ServerPasswordSet | サーバー管理パスワード設定 | ✅ | [x] | 初回接続時ダイアログおよび設定画面にて対応 |
| ServerCertGet | サーバー証明書 (公開鍵) の取得 | ✅ | [ ] | 未実装 (旧ドラフトで `[x]` と誤記されていたが `client.go` に該当コードなし) |
| ServerKeyGet | サーバー証明書の秘密鍵取得 | △ | [ ] | 秘密鍵をローカルに出力する操作のため、実装時は保存先の安全性に注意 |
| ServerCertSet | サーバー証明書・秘密鍵の設定 | ✅ | [ ] | 未実装 (旧ドラフトで `[x]` と誤記されていたが `client.go` に該当コードなし) |
| ServerCertRegenerate | 指定 CN で自己署名証明書を再生成 | ✅ | [ ] | 未着手 |
| ServerCipherGet / ServerCipherSet | 暗号化アルゴリズム取得/設定 | ✅ | [x] | クライアント実装完了 |
| Debug | デバッグコマンド実行 | ✕ | [ ] | サポート/開発者向け専用のため対象外 |
| Crash | サーバープロセスを強制クラッシュさせ再起動 | ✕ | [ ] | 破壊的すぎるため対象外 |
| Flush | 設定をディスクへ即時書き込み | △ | [ ] | vpncmd が設定変更時に自動的にディスク保存するため手動実行の必要性は低い |
| Reboot | VPN Server サービスの再起動 | △ | [ ] | 意図しない切断を防止するためTUI直操作からは除外 (旧ドラフトでは誤って `RebootServer` と記載) |
| ConfigGet / ConfigSet | サーバー設定全体のテキストエクスポート/インポート | △ | [ ] | 設定ファイルの直接編集・書き戻しはCLI/ファイル操作を推奨 (旧ドラフトでは誤って `GetConfig`/`SetConfig` と記載) |

### 1.3 リスナー管理

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| ListenerList | リスナー一覧取得 | ✅ | [x] | リスナー画面 `l` で表示 |
| ListenerCreate | リスナー作成 | ✅ | [x] | リスナー画面 `a` キーで作成 |
| ListenerDelete | リスナー削除 | ✅ | [x] | リスナー画面 `d` キーで削除 |
| ListenerEnable | リスナー有効化 | ✅ | [x] | リスナー画面 `o` キーで有効化 |
| ListenerDisable | リスナー無効化 | ✅ | [x] | リスナー画面 `f` キーで無効化 |

### 1.4 接続維持・TCP接続管理

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| KeepEnable | インターネット接続維持機能の有効化 | ✅ | [ ] | 未着手 (公式マニュアルに記載があるが旧ドラフトでは完全に欠落していた) |
| KeepDisable | インターネット接続維持機能の無効化 | ✅ | [ ] | 同上 |
| KeepSet | 接続維持のホスト・ポート・プロトコル設定 | ✅ | [ ] | 同上 |
| KeepGet | 接続維持の現在設定取得 | ✅ | [ ] | 同上 |
| ConnectionList | サーバーへの現在のTCP接続一覧 | ✅ | [ ] | 未着手。Session (VPN接続) とは別の低レベルTCP接続一覧 |
| ConnectionGet | 個別TCP接続の詳細取得 | ✅ | [ ] | 未着手 |
| ConnectionDisconnect | TCP接続の強制切断 | ✅ | [ ] | 未着手 |

### 1.5 クラスタ・ライセンス管理 (Enterprise/Cluster Edition 向け)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| ClusterSettingGet | 現在のクラスタ設定取得 | △ | [ ] | Enterprise版専用機能 |
| ClusterSettingStandalone | スタンドアロンモードに設定 | △ | [ ] | 同上 (旧ドラフトでは `ClusterSettingSet` という誤った単一コマンド名で記載) |
| ClusterSettingController | クラスタコントローラとして設定 | △ | [ ] | 同上 |
| ClusterSettingMember | クラスタメンバーとして設定 (コントローラ指定) | △ | [ ] | 同上 |
| ClusterMemberList | クラスタメンバー一覧 | △ | [ ] | Enterprise版専用機能 |
| ClusterMemberInfoGet | クラスタメンバーの詳細情報取得 | △ | [ ] | 未着手 (旧ドラフトで欠落) |
| ClusterMemberCertGet | クラスタメンバーの証明書取得 | △ | [ ] | 未着手 (旧ドラフトで欠落) |
| ClusterConnectionStatusGet | クラスタコントローラへの接続状態確認 | △ | [ ] | Enterprise版専用機能 |
| LicenseList / LicenseAdd / LicenseDel / LicenseStatusGet | ライセンスキー管理 | — | [ ] | 要確認: 現行の公式マニュアル 6.3 に掲載が見当たらない。旧バージョン限定または非現行コマンドの可能性 |

### 1.6 ローカルブリッジ・Layer3スイッチ (Router)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| BridgeDeviceList | ブリッジ可能な物理 NIC/tap 一覧 | ✅ | [x] | ブリッジ画面で自動参照 |
| BridgeList | ローカルブリッジ一覧 | ✅ | [x] | ブリッジ画面 `b` で一覧表示 |
| BridgeCreate | ローカルブリッジ作成 | ✅ | [x] | ブリッジ画面 `a` キーで作成 |
| BridgeDelete | ローカルブリッジ削除 | ✅ | [x] | ブリッジ画面 `d` キーで削除 |
| RouterList | 仮想 Layer3 スイッチ一覧 | △ | [ ] | 未着手 (旧ドラフトでは誤って `Layer3List` と記載) |
| RouterAdd / RouterDelete | 仮想 Layer3 スイッチ作成/削除 | △ | [ ] | 同上 (旧: `Layer3AddIf`/`Layer3DelIf` は別物、命名体系が異なる) |
| RouterStart / RouterStop | 仮想 Layer3 スイッチ開始/停止 | △ | [ ] | 未着手 (旧ドラフトには対応するコマンドの記載がなかった) |
| RouterIfList / RouterIfAdd / RouterIfDel | Layer3スイッチのインターフェース一覧/追加/削除 | △ | [ ] | 未着手 (旧: 誤って `Layer3AddIf`/`Layer3DelIf` と記載) |
| RouterTableList / RouterTableAdd / RouterTableDel | Layer3スイッチのルーティングテーブル一覧/追加/削除 | △ | [ ] | 未着手 (旧: 誤って `Layer3AddRoute`/`Layer3DelRoute` と記載) |

### 1.7 ログファイル (サーバー全体)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| LogFileList | サーバー上のログファイル一覧 | △ | [ ] | TUI内での大容量ログ閲覧は負荷が高いため、ローカルログファイル参照を推奨。カテゴリを Hub 管理 (旧2.8) からサーバー全体 (6.3) に修正 |
| LogFileGet | ログファイルのダウンロード | △ | [ ] | 同上 |

### 1.8 リモートアクセスプロトコル拡張

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| IPsecEnable | IPsec/L2TP サーバー機能の有効化/無効化・事前共有鍵設定 | ✅ | [ ] | 未着手 (旧ドラフトで完全に欠落) |
| IPsecGet | IPsec/L2TP サーバー設定取得 | ✅ | [ ] | 未着手 |
| EtherIpClientAdd / EtherIpClientDelete / EtherIpClientList | EtherIP/L2TPv3 クライアント接続設定 | △ | [ ] | レガシープロトコル。未着手 |
| OpenVpnEnable | OpenVPN 互換サーバー機能の有効化/無効化 | ✅ | [ ] | 未着手 |
| OpenVpnGet | OpenVPN サーバー設定取得 | ✅ | [ ] | 未着手 |
| OpenVpnMakeConfig | OpenVPN クライアント設定ファイルのサンプル生成 | ✅ | [ ] | 未着手 |
| SstpEnable / SstpGet | Microsoft SSTP 互換機能の有効化/無効化・設定取得 | ✅ | [ ] | 未着手 |
| VpnOverIcmpDnsGet / VpnOverIcmpDnsEnable | ICMP/DNS 経由 VPN の状態取得/有効・無効化 | ✅ | [x] | クライアント実装完了 |
| DynamicDnsGetStatus | Dynamic DNS 機能の状態確認 | ✅ | [ ] | 未着手 |
| DynamicDnsSetHostname | Dynamic DNS ホスト名設定 | ✅ | [ ] | 未着手 |
| VpnAzureGetStatus | VPN Azure 機能の状態確認 | △ | [ ] | SoftEther提供のクラウド中継サービス依存機能。未着手 |
| VpnAzureSetEnable | VPN Azure 機能の有効化/無効化 | △ | [ ] | 同上 |

### 1.9 syslog・その他

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| SyslogGet / SyslogEnable / SyslogDisable | syslog 転送設定の取得/有効・無効化 | ✅ | [x] | クライアント実装完了 |

### 1.10 Virtual HUB 作成・削除 (サーバー側から)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| HubList | Hub 一覧取得 | ✅ | [x] | ダッシュボードに一覧表示 |
| HubCreate | Hub 作成 (スタンドアロン) | ✅ | [x] | ダッシュボード `a` キーで作成 |
| HubCreateDynamic | 動的 Hub 作成 (クラスタ用) | △ | [ ] | クラスタ環境向け機能。スタンドアロン Hub 作成で代替 |
| HubCreateStatic | 静的 Hub 作成 (クラスタ用) | △ | [ ] | 同上 |
| HubDelete | Hub 削除 | ✅ | [x] | ダッシュボード `d` キーで削除 |
| HubSetStatic / HubSetDynamic | Hub種別 (静的/動的) の変更 | △ | [ ] | クラスタ環境向け機能。未着手 (旧ドラフトで欠落) |
| Hub | 指定 Hub を選択し Hub 管理モードへ遷移 | ✅ | [~] | TUI タブ切替時に `/HUB:<name>` オプションでシームレス遷移 (対話的な `Hub` コマンド自体は使用していない) |

---

## 2. Virtual HUB 管理コマンド (Hub Management Mode、6.4)

`Hub <名前>` で対象 Hub を選択した後に実行できるコマンド群 (本アプリでは `/HUB:<name>` 接続オプションで同等の効果を得ている)。

### 2.1 Hub 基本操作

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| Online | 選択中 Hub をオンライン化 | ✅ | [x] | Hub画面 `o` キー |
| Offline | 選択中 Hub をオフライン化 | ✅ | [x] | Hub画面 `f` キー |
| SetMaxSession | 最大接続セッション数設定 | ✅ | [x] | クライアント実装完了 |
| SetHubPassword | Hub 管理パスワード設定 | ✅ | [x] | ダッシュボード `p` キーで設定プロンプト起動 |
| SetEnumAllow / SetEnumDeny | 匿名ユーザーによるHub列挙許可/拒否 | ✅ | [x] | クライアント実装完了 |
| OptionsGet | Hub 動作オプション取得 | ✅ | [x] | Hub概要画面で自動表示 |
| StatusGet | Hub 詳細状態取得 | ✅ | [x] | Hub 概要タブで全自動表示 |

### 2.2 ユーザー管理 (重点機能)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| UserList | ユーザー一覧取得 | ✅ | [x] | Usersタブで自動取得・検索対応 |
| UserCreate | ユーザー作成 | ✅ | [x] | Usersタブ `a`/`c` キーで作成 |
| UserGet | ユーザー詳細取得 | ✅ | [x] | ユーザー詳細画面でインライン編集対応 |
| UserSet | ユーザー詳細変更 | ✅ | [x] | グループ・氏名・備考などのインライン編集 |
| UserDelete | ユーザー削除 | ✅ | [x] | Usersタブ `d` キーで削除 |
| UserAnonymousSet | 匿名認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserPasswordSet | パスワード認証への設定/再設定 | ✅ | [x] | ユーザー詳細画面で `p` キー再設定 |
| UserCertSet | X.509証明書認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserCertGet | ユーザー証明書の取得 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| UserSignedSet | 署名付き証明書認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserRadiusSet | RADIUS 認証への変更 | ✅ | [x] | クライアント実装完了 |
| UserNTLMSet | NTLM (Windowsドメイン) 認証への変更 | △ | [x] | クライアント実装完了。ADドメイン参加環境が前提 |
| UserPolicySet | ユーザーポリシー (帯域制限・接続制限) 設定 | ✅ | [x] | クライアント実装完了 |
| UserPolicyRemove | ユーザーポリシー削除 | ✅ | [x] | クライアント実装完了 |
| UserExpiresSet | アカウント有効期限設定 | ✅ | [x] | ユーザー詳細画面でインライン変更 |
| PolicyList | 設定可能なセキュリティポリシー項目一覧 | ✅ | [ ] | 未実装 (旧ドラフトで `[x]` と誤記されていたが `client.go` に該当コードなし) |

### 2.3 グループ管理

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| GroupList | グループ一覧取得 | ✅ | [x] | Groupsタブで一覧表示 |
| GroupCreate | グループ作成 | ✅ | [x] | Groupsタブ `a`/`c` キーで作成 |
| GroupGet | グループ詳細取得 | ✅ | [x] | グループ詳細画面で表示・編集 |
| GroupSet | グループ詳細変更 | ✅ | [x] | 氏名・備考のインライン編集対応 |
| GroupDelete | グループ削除 | ✅ | [x] | Groupsタブ `d` キーで削除 |
| GroupJoin | ユーザーをグループに追加 | ✅ | [~] | `UserSet /GROUP:` で同等の効果を実現済み。専用コマンドとしては未実装 |
| GroupUnjoin | ユーザーをグループから除外 | ✅ | [~] | 同上 (`UserSet /GROUP:` の空値指定で対応) |
| GroupPolicySet | グループポリシー設定 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| GroupPolicyRemove | グループポリシー削除 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |

### 2.4 セッション管理 (重点機能)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| SessionList | 現在の接続セッション一覧 | ✅ | [x] | Sessionsタブでリアルタイム自動更新 |
| SessionGet | セッション詳細取得 | ✅ | [x] | 一覧テーブルの各行データとして統合表示 |
| SessionDisconnect | セッション強制切断 | ✅ | [x] | Sessionsタブ `x` キーで切断 |

### 2.5 MAC / IP テーブル

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| MacTable | Hub内のMACアドレステーブル表示 | △ | [ ] | 未着手 (旧ドラフトで完全に欠落) |
| MacDelete | MACアドレステーブルエントリ削除 | △ | [ ] | 未着手 |
| IpTable | Hub内のIPアドレステーブル表示 | △ | [ ] | 未着手 |
| IpDelete | IPアドレステーブルエントリ削除 | △ | [ ] | 未着手 |

### 2.6 SecureNAT / NAT / DHCP / RADIUS

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| SecureNatEnable / SecureNatDisable | SecureNAT 全体有効化/無効化 | ✅ | [x] | SecureNATタブ `o/f` キー |
| SecureNatStatusGet | SecureNAT 状態取得 | ✅ | [x] | SecureNATタブで表示 |
| SecureNatHostGet / SecureNatHostSet | 仮想ホスト (IP/MAC/Subnet) 設定取得/変更 | ✅ | [x] | SecureNATタブでインライン編集 |
| NatGet | NAT 動作設定取得 | ✅ | [x] | SecureNATタブで表示 |
| NatEnable / NatDisable | Virtual NAT 個別有効化/無効化 | ✅ | [x] | SecureNATタブ `n/N` キー |
| NatSet | NAT 動作設定変更 | ✅ | [ ] | 未実装 (旧ドラフトで `[x]` と誤記されていたが `client.go` に該当コードなし。設定変更は `SecureNatHostSet` に統合する方針だったが未実装のまま) |
| NatTable | NATセッションテーブル表示 | △ | [ ] | 未着手 (旧ドラフトで欠落) |
| DhcpGet / DhcpSet | DHCP 配布範囲・リース時間設定取得/変更 | ✅ | [x] | SecureNATタブでインライン編集 |
| DhcpEnable / DhcpDisable | Virtual DHCP 個別有効化/無効化 | ✅ | [x] | SecureNATタブ `h/H` キー |
| DhcpTable | DHCPリーステーブル表示 | △ | [ ] | 未着手 (旧ドラフトで欠落) |
| RadiusServerGet / RadiusServerSet / RadiusServerDelete | RADIUS サーバー設定取得/変更/削除 | ✅ | [x] | Hub概要 `R` キーでモーダル設定 |

### 2.7 アクセスリスト (パケットフィルタ)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AccessList | アクセスリスト一覧取得 | ✅ | [x] | ACLタブで表示 |
| AccessAdd | アクセスリストルール追加 (IPv4) | ✅ | [x] | ACLタブ `a`/`c` キーでフォーム追加 |
| AccessAddEx | 遅延/ジッタ/パケットロス設定付きルール追加 (IPv4) | △ | [ ] | 未着手 (旧ドラフトで欠落) |
| AccessAdd6 | アクセスリストルール追加 (IPv6) | ✅ | [ ] | 未着手 (旧ドラフトで欠落。IPv4のみ対応) |
| AccessAddEx6 | 遅延/ジッタ/パケットロス設定付きルール追加 (IPv6) | △ | [ ] | 未着手 |
| AccessDelete | アクセスリストルール削除 | ✅ | [x] | ACLタブ `d` キーで削除 |
| AccessEnable / AccessDisable | ルール有効化/無効化 | ✅ | [x] | ACLタブ `o/f` キーで切替 |

### 2.8 接続元IP制限リスト (`Ac*`、AccessListとは別機能)

> `AccessList` (パケットフィルタ、2.7) とは別の、Hubへの接続そのものを送信元IPで許可/拒否する機能。旧ドラフトでは完全に見落とされていた。

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AcList | 接続元IP制限リスト一覧 | ✅ | [ ] | 未着手 (旧ドラフトで完全に欠落) |
| AcAdd | 接続元IP制限ルール追加 (IPv4) | ✅ | [ ] | 未着手 |
| AcAdd6 | 接続元IP制限ルール追加 (IPv6) | ✅ | [ ] | 未着手 |
| AcDel | 接続元IP制限ルール削除 | ✅ | [ ] | 未着手 |

### 2.9 カスケード接続 (拠点間接続、Bridge 向け)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CascadeList | カスケード接続一覧 | ✅ | [x] | Cascadeタブで表示 |
| CascadeCreate | カスケード接続作成 | ✅ | [x] | Cascadeタブ `a`/`c` キーで作成 |
| CascadeGet / CascadeSet | カスケード接続詳細設定取得/変更 | ✅ | [x] | 作成プロンプトおよび一覧で管理 |
| CascadeDelete | カスケード接続削除 | ✅ | [x] | Cascadeタブ `d` キーで削除 |
| CascadeStatusGet / CascadeDetailGet | カスケード接続状態・詳細取得 | ✅ | [x] | クライアント実装完了 |
| CascadeOnline / CascadeOffline | オンライン/オフライン化切替 | ✅ | [x] | Cascadeタブ `o/f` キーで切替 |
| CascadeRename | カスケード接続名の変更 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| CascadeUsernameSet | カスケード認証ユーザー名設定 | ✅ | [ ] | 未着手 |
| CascadeAnonymousSet | カスケードを匿名認証に変更 | ✅ | [ ] | 未着手 |
| CascadePasswordSet | カスケードをパスワード認証に変更 | ✅ | [x] | クライアント実装完了 |
| CascadeCertSet / CascadeCertGet | カスケードのクライアント証明書認証設定/取得 | ✅ | [ ] | 未着手 |
| CascadeEncryptEnable / CascadeEncryptDisable | カスケード接続のSSL暗号化有効/無効化 | ✅ | [ ] | 未着手 |
| CascadeCompressEnable / CascadeCompressDisable | カスケード接続のデータ圧縮有効/無効化 | ✅ | [ ] | 未着手 |
| CascadeProxyNone / CascadeProxyHttp / CascadeProxySocks | カスケード接続のプロキシ経由設定 (直接/HTTP/SOCKS) | △ | [ ] | 未着手 |
| CascadeServerCertEnable / CascadeServerCertDisable | 接続先サーバー証明書検証の有効/無効化 | ✅ | [ ] | 未着手 |
| CascadeServerCertSet / CascadeServerCertDelete / CascadeServerCertGet | 接続先サーバー証明書の登録/削除/取得 (証明書ピン留め) | ✅ | [ ] | 未着手。app_specs.md 7章のセキュリティ方針と関連 |
| CascadeDetailSet | カスケード接続の詳細プロトコル設定 | △ | [ ] | 未着手 |
| CascadePolicySet | カスケード接続へのセキュリティポリシー適用 | ✅ | [ ] | 未着手 |

### 2.10 CA証明書管理 (信頼するCA一覧)

> Hub単位で信頼するCA証明書を管理する機能。旧ドラフトでは完全に欠落していた。

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CAList | 信頼するCA証明書一覧取得 | ✅ | [ ] | 未着手 (旧ドラフトで完全に欠落) |
| CAAdd | 信頼するCA証明書の追加 | ✅ | [ ] | 未着手 |
| CADelete | 信頼するCA証明書の削除 | ✅ | [ ] | 未着手 |
| CAGet | 信頼するCA証明書の取得 (ファイル出力) | ✅ | [ ] | 未着手 |

### 2.11 証明書失効リスト (CRL)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CrlList | CRL 登録一覧 | △ | [ ] | PKI・電子証明書運用環境専用 |
| CrlAdd | CRL 追加 | △ | [ ] | 同上 |
| CrlDel | CRL 削除 | △ | [ ] | 同上 |
| CrlGet | CRL エントリの取得 | △ | [ ] | 未着手 (旧ドラフトで欠落) |

### 2.12 Hub 拡張オプション

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AdminOptionList | Hub管理オプション一覧取得 | ✅ | [ ] | 未着手 (旧ドラフトで完全に欠落) |
| AdminOptionSet | Hub管理オプション設定 | ✅ | [ ] | 未着手 |
| ExtOptionList | Hub拡張オプション一覧取得 | ✅ | [ ] | 未着手 |
| ExtOptionSet | Hub拡張オプション設定 | ✅ | [ ] | 未着手 |

### 2.13 セキュリティログ / パケットログ

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| LogGet | ログ設定取得 | ✅ | [x] | Logタブで表示 |
| LogEnable / LogDisable | セキュリティ/パケットログ有効化/無効化 | ✅ | [x] | Logタブで Enter/Space 切替 |
| LogPacketSaveType | パケットログ保存種別設定 (DHCP/TCP/UDP等) | ✅ | [x] | Logタブで Enter/Space 切替 |
| LogSwitchSet | ログファイル切り替え周期設定 | ✅ | [x] | Logタブで Enter/Space 切替 |

> ログファイル本体の一覧・ダウンロード (`LogFileList`/`LogFileGet`) はサーバー全体コマンドのため 1.7 章に移動した。

---

## 3. VPN Bridge 管理コマンド (`/BRIDGE` モード)

VPN Bridge は VPN Server のサブセット実装であり、コマンド体系は 1章・2章と共通。ローカルブリッジおよびカスケード接続操作は Server モードと共通のTUIで操作可能。

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| BridgeList / BridgeCreate / BridgeDelete | ローカルブリッジ一覧・作成・削除 | ✅ | [x] | Serverモードと共通画面で操作可能 |
| CascadeList / CascadeCreate / CascadeDelete / CascadeOnline / CascadeOffline | カスケード接続一覧・作成・切断等 | ✅ | [x] | Serverモードと共通画面で操作可能 |
| ListenerList / ListenerCreate / ListenerDelete | リスナー管理 (1.3 と共通) | ✅ | [x] | Serverモードと共通画面で操作可能 |
| ServerCipherGet / ServerCipherSet | 暗号化アルゴリズム管理 (1.2 と共通) | ✅ | [x] | Serverモードと共通画面で操作可能 |
| ServerCertGet / ServerCertSet | 証明書管理 (1.2 と共通) | ✅ | [ ] | 1.2 と同様、未実装 |

`/BRIDGE` モードでの動作は単体テストで `Target.Mode=ModeBridge` の場合に全コマンドが `/SERVER` ではなく `/BRIDGE` フラグを使うことを確認済み。実際の VPN Bridge サーバーでの動作確認は未実施。

---

## 4. VPN Client 管理コマンド (`/CLIENT` モード)

> 当初は将来ロードマップだったが M7 で前倒し着手 (app_specs.md 5.10 参照)。

### 4.1 バージョン・サービス設定

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| About | vpncmd 自体のバージョン情報表示 | ✅ | [~] | TUI 自身のバージョンを表示 (1.1 と同様) |
| VersionGet | VPN Client サービスのバージョン情報取得 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| PasswordSet | VPN Client サービスへの接続用パスワード設定 | ✅ | [ ] | 未着手。Account の接続パスワードとは別物 (サービス自体の管理パスワード) |
| PasswordGet | VPN Client サービスのパスワード設定状態取得 | ✅ | [ ] | 未着手 |
| RemoteEnable / RemoteDisable | VPN Client サービスのリモート管理許可/禁止 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |

### 4.2 信頼するCA証明書・スマートカード

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| CertList | 信頼するCA証明書一覧取得 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| CertAdd | 信頼するCA証明書の追加 | ✅ | [ ] | 未着手 |
| CertDelete | 信頼するCA証明書の削除 | ✅ | [ ] | 未着手 |
| CertGet | 信頼するCA証明書の取得 | ✅ | [ ] | 未着手 |
| SecureList | 利用可能なスマートカード種類の一覧取得 | △ | [ ] | 専用ハードウェア依存のため優先度低 |
| SecureSelect | 使用するスマートカード種類の選択 | △ | [ ] | 同上 |
| SecureGet | 現在選択中のスマートカード種類取得 | △ | [ ] | 同上 |

### 4.3 仮想 NIC 管理

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| NicList | 仮想 LAN カード一覧の取得 | ✅ | [x] | Client画面で表示 |
| NicCreate | 新規仮想 LAN カードの作成 | ✅ | [x] | Client画面で作成 |
| NicDelete | 仮想 LAN カードの削除 | ✅ | [x] | Client画面で削除 |
| NicUpgrade | 仮想 NIC デバイスドライバの更新 | △ | [ ] | OS依存の管理者権限操作。未着手 |
| NicGetSetting / NicSetSetting | 仮想 NIC のMACアドレス設定取得/変更 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| NicEnable / NicDisable | 仮想 NIC の有効化/無効化 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |

### 4.4 接続設定 (アカウント) 管理

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| AccountList | 接続設定一覧の取得 | ✅ | [x] | Client画面で表示 |
| AccountCreate | 新しい接続設定の作成 | ✅ | [x] | Client画面 `a`/`c` キーで作成 |
| AccountGet | 接続設定の詳細取得 | ✅ | [x] | クライアント実装完了 |
| AccountDetailGet | 接続設定の詳細プロトコル情報取得 | ✅ | [~] | クライアント実装済みだが、公式マニュアルには `AccountDetailSet` のみ記載があり `AccountDetailGet` の実在は要確認 |
| AccountSet | 接続先・Hub設定の変更 | ✅ | [x] | クライアント実装完了 |
| AccountDelete | 接続設定の削除 | ✅ | [x] | Client画面 `d` キーで削除 |
| AccountUsernameSet | 接続認証用ユーザー名の設定 | ✅ | [x] | Client画面 `u` キーでプロンプト変更対応 |
| AccountAnonymousSet | 匿名認証への変更 | ✅ | [x] | クライアント実装完了 |
| AccountPasswordSet | パスワード認証設定/変更 | ✅ | [x] | Client画面 `p` キーでプロンプト変更対応 |
| AccountCertSet / AccountCertGet | クライアント証明書認証への変更/証明書取得 | △ | [ ] | クライアントPKI証明書指定プロンプト未実装 |
| AccountEncryptEnable / AccountEncryptDisable | SSL暗号化の有効/無効化 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| AccountCompressEnable / AccountCompressDisable | データ圧縮の有効/無効化 | ✅ | [ ] | 未着手 |
| AccountProxyNone / AccountProxyHttp / AccountProxySocks | プロキシ経由接続設定 (直接/HTTP/SOCKS) | △ | [ ] | 未着手 |
| AccountServerCertEnable / AccountServerCertDisable | 接続先サーバー証明書検証の有効/無効化 | ✅ | [ ] | 未着手 |
| AccountServerCertSet / AccountServerCertDelete / AccountServerCertGet | 接続先サーバー証明書の登録/削除/取得 (証明書ピン留め) | ✅ | [ ] | 未着手。app_specs.md 7章のセキュリティ方針と関連 |
| AccountDetailSet | 詳細プロトコル設定 (MTU/半二重等) | △ | [ ] | 未着手 |
| AccountRename | 接続設定名の変更 | ✅ | [ ] | 未着手 (旧ドラフトで欠落) |
| AccountNicSet | 接続設定に使用する仮想NICの指定 | ✅ | [ ] | 未着手 |
| AccountStatusShow / AccountStatusHide | 接続時のステータス表示有効/無効化 | △ | [ ] | GUI向けオプションのためTUIでは優先度低 |
| AccountSecureCertSet | スマートカード認証の設定 | △ | [ ] | 専用ハードウェア依存 |
| AccountRetrySet | 再接続試行回数・間隔の設定 | ✅ | [ ] | 未着手 |
| AccountStartupSet / AccountStartupRemove | 自動起動接続の設定/解除 | ✅ | [ ] | 未着手 (OS起動時の自動接続、意図的に見送り) |
| AccountExport / AccountImport | 接続設定のエクスポート/インポート | ✅ | [ ] | 未着手 |
| AccountConnect / AccountDisconnect | 接続開始/切断 | ✅ | [x] | Client画面 `o` / `f` キーで操作 |
| AccountStatusGet | 接続状態・統計取得 | ✅ | [x] | クライアント実装完了 |

### 4.5 接続維持 (Client側)

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| KeepEnable / KeepDisable | インターネット接続維持機能の有効化/無効化 | ✅ | [ ] | 未着手 (旧ドラフトで欠落。1.4 のサーバー版とは別実装が必要) |
| KeepSet / KeepGet | 接続維持のホスト・ポート・プロトコル設定/取得 | ✅ | [ ] | 未着手 |

---

## 5. VPN Tools コマンド (`/TOOLS` モード)

> サーバー/クライアント接続 (`/SERVER`, `/HUB`, `/CLIENT`) を伴わないスタンドアロンユーティリティ群。オフライン処理が中心のため、TUIとしての価値は他モードより低いが、方針上は対応を検討する。

| コマンド | 概要 | 対応方針 | 実装状況 | 備考・未実装の理由 |
|---|---|---|---|---|
| About | vpncmd 自体のバージョン・ビルド情報表示 | △ | [ ] | CLI直接実行、またはTUI自身の`--version`表示で代替 |
| Check | 動作環境チェック (TUN/TAP・メモリ・ネットワーク依存性) | △ | [ ] | OS環境チェック用診断ユーティリティ。結果をTUI上に表示する形で対応検討 |
| MakeCert | X.509証明書・秘密鍵の新規生成 (RSA 1024bit、自己署名/CA署名両対応) | △ | [ ] | 鍵生成結果のファイル保存を伴う。旧ドラフトの `MakeCert2048` は公式マニュアルに記載がなく誤記の可能性 |
| TrafficClient / TrafficServer | 通信スループット性能測定ツール (クライアント/サーバー) | △ | [ ] | 継続的な双方向パケットトラフィック生成・ストリーミング処理のためTUI化の優先度は低い |
| PortChecker | ポート到達性チェック | ✕ | [ ] | 現行の公式マニュアルに記載なし。廃止済み (存在しないコマンド) と判断 |

---

## 更新方針

- 実装を進めるたびに「実装状況」列を更新する。実装確認は `internal/vpncmd/client.go` 内の実際のコマンド送信箇所を根拠とし、コメントや推測で `[x]` にしない。
- 「要確認」の項目は実装着手時に正式なコマンド名・パラメータを確認し、確定したら注記を削除する。
- 存在しないことが判明したコマンドは打ち消し線を引くか行を削除し、新たに判明したコマンドは追記する。
- 公式マニュアルが改訂された場合は本ファイルの典拠URLも合わせて見直す。
