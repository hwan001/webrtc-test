
### key
```bash
# 개인 키 생성
openssl genpkey -algorithm RSA -out server-key.pem -pkeyopt rsa_keygen_bits:2048

# 자체 서명된 인증서 생성
openssl req -new -x509 -key server-key.pem -out server-cert.pem -days 365 -subj "/CN=localhost"
```

### curl (http3)
- https://blog.joe-brothers.com/macos-curl-with-http3/
```bash
# 기존에 설치된 curl 제거
brew remove -f curl

# cloudflare가 제공하는 curl ruby script 다운로드
wget https://raw.githubusercontent.com/cloudflare/homebrew-cloudflare/master/curl.rb

# curl 설치
brew install --formula --HEAD -s curl.rb

# 설치한 curl PATH에 추가
echo 'export PATH="$(brew --prefix)/opt/curl/bin:$PATH"' >> ~/.zshrc

# 쉘 재시작

# PATH 제대로 수정되었는지 확인
which curl # Apple silicon의 경우, "/opt/homebrew/opt/curl/bin/curl"

# http3 Feature enable 되었는지 확인
curl --version | grep HTTP3
  Features: alt-svc AsynchDNS brotli HTTP2 HTTP3 IDN IPv6 Largefile libz MultiSSL NTLM NTLM_WB SSL UnixSockets zstd

# http3 지원하는 서버에 요청해서 테스트
curl --http3 https://cloudflare-quic.com -I
# 아래처럼 나오는지 확인
HTTP/3 200
date: Sat, 17 Aug 2024 08:09:01 GMT
content-type: text/html
content-length: 125959
server: cloudflare
cf-ray: 8b482dfb4bae29e3-FUK
alt-svc: h3=":443"; ma=86400
```

### http3 curl test
```bash
curl --http3 https://localhost -I -k
```