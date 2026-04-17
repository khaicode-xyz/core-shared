cd /Users/Ann/ECQ/Core/core-shared

-- add
git init
git add README.md
git commit -m "first commit"
git branch -M main
git remote add origin https://github.com/khaicode-xyz/core-shared.git
git push -u origin main

-- update
git remote add origin https://github.com/khaicode-xyz/core-shared.git
git branch -M main
git push -u origin main

-- tag
git tag v0.1.0
git push origin v0.1.0

-- commit
cd core-shared && git add . && git commit -m "Add HTTPClient + DialGRPC base helpers" && git tag v0.2.0 && git push origin main v0.2.0

-- service
go get github.com/khaicode-xyz/core-shared@v0.2.0 && go mod tidy