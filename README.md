Form2Mail
---------

A small server to receive form inputs and then send them to specific email inboxes via Aliyun DirectMail.

```
go get -v -u github.com/caiguanhao/Form2Mail
./Form2Mail \
  --akid 0000000000000000 --aksecret 000000000000000000000000000000 \
  --from mail@example.com --alias YourName \
  --subject "New User Comment" --to myemail@gmail.com \
  --listen "127.0.0.1:8080"
```
