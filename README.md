# mTLS ワークショップ

# はじめに

この文書は実際にローカルPC上で、mTLSを体験するためのものです。

*mTLS* は Mutual TLS(Transport Layer Security) の略です。
これはTLS 1.3について書かれたRFC8446の中で、Client Authentication として説明されています。

The Transport Layer Security (TLS) Protocol Version 1.3
https://datatracker.ietf.org/doc/html/rfc8446

ですが、このRFCを読んで理解するには要求される前提知識も多く難しいです。
周辺知識も含めて理解したいのであれば、[Bulletproof TLS and PKI](https://amzn.to/3UQoCOm)(邦訳 [プロフェッショナルTLS＆PKI](https://amzn.to/3Ww92sm))を読むことをお勧めします。

これらの本の該当箇所を読んでから、この文書の一連の手順を追うとより理解が進むでしょう。

# 準備

以下のコマンドを要求します。

- openssl
- curl
- go
    - テストサーバー実行のため

# 用語

事前にいくつかの用語を紹介します。


- 公開鍵暗号方式

    簡単に言うと、暗号化のための鍵(公開鍵)と暗号を解除するための鍵(秘密鍵)が分かれている方式です。
    この性質を持った暗号アルゴリズムはいくつも開発されています。

- 電子署名

    公開鍵暗号方式の仕組みを使って、電磁記録が改竄されていないことを示す仕組みです。
    これもいろいろなアルゴリズムがあります。
    電子署名自体は誰でも任意の電磁記録(ファイル)に対して実行可能です。

- デジタル証明書

    電子署名が間違いなくその本人によって行われたことを示すために使われます。


- 認証局(CA, Certificate Authorities)

    証明書を作るためのサービスです。
    認証局は自身の秘密鍵と証明書を持ちます。
    認証局自体は誰でも構築出来るのですが、認証局自体が信頼できるかは別問題です。
    この実験ではサーバー証明書用認証局とクライアント証明書認証局をそれぞれ作ります。

- サーバー証明書

    電子署名技術の応用の一つです。不特定多数のクライアントに対して、サーバー(ドメイン)管理者の存在を証明するために使われます。
    サーバー証明書は、サーバー管理者がパブリックな認証局に依頼して発行してもらいます。
    今回は実験なのでプライベート認証局によるサーバー証明書を使います。

- クライアント証明書

    電子署名技術の応用の一つです。サーバー管理者が自身のサービスに接続するクライアントが既知のクライアントであることを証明するために用います。
    クライアント証明書はサーバー管理者が特定のクライアントを認証するために使うので、サーバー管理者によって運用されるプライベート認証局による証明書が使われます。

# 証明書を作る基本的な手順

証明書を作るのは手順さえわかっていれば簡単です。
サーバー証明書もクライアント証明書も以下のステップで実行されます。

1. 証明書を必要とする人が、CSR(Certificate Signing Request)を作る
2. 認証局が、自身の秘密鍵を使ってCSRから新しい証明書を作る

サーバー証明書が必要な場合、CSRを作るのはサーバー管理者です。
クライアント証明書が必要な場合、CSRを作るのはクライアントサービスの管理者です。
今回の実験では認証局自体も自分で作成しますが、クラウドサービスが提供するプライベート認証局機能というものもあります。

# 実験用サーバー準備

まずは実験用HTTPSサーバーを立ち上げます。
繰り返しますが、実際のサーバー証明書は通常自分で発行するのではなく、既存の適切なパブリック認証局を使います。
今回は実験のために、あえてサーバー証明書も自分自身で発行します。

## Step1: (オレオレ)サーバー証明書認証局を作る

認証局を作るということをもっとも簡単に言い表すと、具体的には認証局としての秘密鍵と証明書を作ることを指します。
認証局の証明書はより上位の認証局から発行してもらうのが普通ですが、今回は自己署名を行って証明書を作ります。
つまりプライベートルート認証局です。

### Subject

ところで、証明書にはSubjectという証明書の主体者を示すフィールドがあります。
どういう項目を入れるかについては一応ルールが存在します。

https://datatracker.ietf.org/doc/html/rfc4514

```
String  X.500 AttributeType
------  --------------------------------------------
CN      commonName (2.5.4.3)
L       localityName (2.5.4.7)
ST      stateOrProvinceName (2.5.4.8)
O       organizationName (2.5.4.10)
OU      organizationalUnitName (2.5.4.11)
C       countryName (2.5.4.6)
STREET  streetAddress (2.5.4.9)
DC      domainComponent (0.9.2342.19200300.100.1.25)
UID     userId (0.9.2342.19200300.100.1.1)
```

実験用サーバー証明書認証局のSubjectは以下のようにしてみます。

`/C=JP/ST=Tokyo/L=Chuo/O=Kame/CN=ca.server.example.com`


### 認証局作成コマンド

```sh
cd server-ca
openssl req -x509 -newkey ec:<(openssl ecparam -name secp384r1) \
-keyout server-ca.key \
-out server-ca.crt \
-days 365 \
-subj "/C=JP/ST=Tokyo/L=Chuo/O=Kame/CN=ca.server.example.com" \
-nodes
# 証明書内容確認
openssl x509 -in server-ca.crt -noout -pubkey -subject
```

## Step2: サーバー証明書を発行する

認証局ができたので、次にサーバー証明書を作りましょう。
サーバー証明書を必要とするのは認証局の人ではなく、サーバー管理者となります。
このステップは、どの立場の人が操作を行なっているのか意識しながら手順を進めてください。
それぞれのステップは以下の通りですが、CSRや証明書を渡す部分は特に記述しません。

1. サーバー管理者が秘密鍵をつくる
2. サーバー管理者がCSRを作成する
3. サーバー管理者がCSRを認証局に渡す
4. 認証局がCSRを認証局の秘密鍵で署名してサーバー証明書を作る
5. 認証局がCSRをサーバー管理者に渡す

しつこく繰り返しますが、サーバー証明書はパブリック認証局に依頼して作成するのが通常です。
実際に利用するサーバー証明書の発行については、利用する認証局サービスの手順にしたがってください。

### Step2-1: サーバーの秘密鍵を発行する

まずはサーバー管理者として、サーバーの秘密鍵を作ります。

```sh
cd server
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:secp384r1 -out server.key
```

### Step2-2: CSR(Certificate Signing Request)

次に秘密鍵からCSRを作ります。これは署名作成依頼のためのファイルです。
つまりこれもサーバー管理者の役割として実施します。
ところで、ここで指定するSubjectはサーバーとしてのものです。
このためCNにはドメインを指定します。今回はlocalhostだけで動かすのでlocalhostです。

```sh
cd server
openssl req -new -key server.key -out server.csr \
-subj "/C=JP/ST=Tokyo/L=Chuo/O=Kame/CN=localhost"
# CSR内容確認
openssl req -in server.csr -text -noout
```

### Step2-3: サーバー証明書発行

ここからあなたはサーバー認証局管理者として行動します。
CSRをCAの秘密鍵を使って署名すればサーバー証明書の発行ができます。

```sh
cd server-ca
openssl x509 -req \
-in ../server/server.csr \
-CA server-ca.crt \
-CAkey server-ca.key \
-CAcreateserial -out ../server/server.crt -days 365
openssl x509 -in ../server/server.crt -noout -pubkey -subject
```

## Step3: HTTPSサーバーを動かす

HTTPSサーバーとして動かすには、サーバー証明書とその秘密鍵が必要です。
サンプルコードを参照してください。

```sh
cd server
go run server-no-client-auth.go
```

## Step4: アクセスする

```sh
cd client
curl http://localhost:8443 # 失敗(httpによるアクセス)
curl https://localhost:8443 # 失敗(証明書が指定されていない)
curl -k server.crt https://localhost:8443 # 証明書エラーを無視するオプションを指定
curl https://localhost:8443 --cacert ../server/server.crt # 証明書を指定
```

これでサーバー証明書の発行とそれを使ったHTTPSサーバーをローカルで立ち上げることができました。

# クライアント証明書を発行する

ここからが実験の肝です。

## Step1: クライアント証明書認証局を作る

クライアント証明書の認証局を作ります。
クライアント証明書の発行はサーバーにアクセスできる人を定義するのと同じです。
つまりクライアント証明書の認証局はサーバー管理者と同じである必要があります。

### クライアント認証局作成コマンド実行

```sh
cd client-ca
openssl req -x509 -newkey ec:<(openssl ecparam -name secp384r1) \
-keyout client-ca.key \
-out client-ca.crt \
-days 365 \
-subj "/C=JP/ST=Tokyo/L=Chuo/O=Kame/CN=ca.client.example.com" \
-nodes
openssl x509 -in client-ca.crt -noout -pubkey -subject
```

証明書の発行の都度、新しい認証局を作る必要はありません。
一つの認証局で複数の証明書を発行できます。

## Step2: クライアント証明書を作る

クライアント証明書の場合もクライアントの秘密鍵を作って、CSRを発行します。
ですが秘密鍵やCSRを作成するのは、クライアントサービスを管理している人の役割です。
作成したCSRはクライアント証明書を発行する認証局 = サーバー管理者に渡されることに注意してください。

1. クライアントサービス管理者が秘密鍵をつくる
2. クライアントサービス管理者がCSRを作成する
3. クライアントサービス管理者がCSRを認証局に渡す
4. 認証局がCSRを認証局の秘密鍵で署名してクライアント証明書を作る
5. 認証局がCSRをクライアントサービス管理者に渡す

### Step2-1 クライアントの秘密鍵を発行

秘密鍵の発行自体は、サーバーだろうとクライアントだろうと変わりません。

```sh
cd client
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:secp384r1 -out client.key
```

### Step2-2 クライアント証明書のためのCSRを作成する。

CSR作成も基本的なコマンドは変わらないですが、クライアントの場合CNがドメイン名である必要はありません。
ここでは `Super Client` としてみました。

```sh
cd client
openssl req -new -key client.key -out client.csr \
-subj "/C=JP/ST=Tokyo/L=Chuo/O=Kame/CN=Super Client"
openssl req -in client.csr -text -noout
```

### Step2-3 クライアント認証局でクライアント証明書を発行する

このステップでは、あなたはCSRを受け取ったクライアント証明書認証局の役割となります。

```sh
cd client-ca
openssl x509 -req \
-in ../client/client.csr \
-CA client-ca.crt \
-CAkey client-ca.key \
-CAcreateserial -out ../client/client.crt -days 365
openssl x509 -in ../client/client.crt -noout -pubkey -subject
```

## Step3: クライアント証明書を要求するHTTPSサーバーを動かす

クライアント証明書を要求するサーバーは、サーバー証明書とその秘密鍵に加えて、クライアント認証局の証明書が必要です。

```sh
cd server
go run server-require-client-auth.go
```

## Step4: クライアント証明書を使ってサーバーにアクセスする。

今回はプライベートサーバー証明書も指定していますが、通常は指定しません。

```sh
cd client
curl --cert client.crt --key client.key https://localhost:9443 --cacert ../server/server.crt
```

サーバーのログとしてクライアント証明書のSubjctが表示されていることを確認してください。

# まとめ

以上がmTLSの基本的な動作となります。
実際に運用するとなると認証局の運用や期限切れへの対応など難しい点は多いです。
ですが認証手段としては非常に強力ですし、アプリケーションとは別のレイヤーで解決されるという特徴もあります。
みなさんの技術選定の助けになれば幸いです。
