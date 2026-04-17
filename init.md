cd /Users/Ann/ECQ/Core/core-shared

git init -b main
git add .
git commit -m "Initial core-shared module with 9 shared packages"

git remote add origin git@github.com:khaicode/core-shared.git
git push -u origin main

git tag v0.1.0
git push origin v0.1.0