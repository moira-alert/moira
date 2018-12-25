USER="skbkontur"
REPO="moira-kontur-senders"

BODY='{
 "request": {
 "message": "Triggered from Moira (https://github.com/moira-alert/moira)",
 "branch":"master",
 "config": {
   "env": {
     "NOTIFIER_BRANCH": $TRAVIS_BRANCH
   }
  }
}}'

curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "Travis-API-Version: 3" \
  -H "Authorization: token ${TRAVIS_API_TOKEN}" \
  -d "$BODY" \
  https://api.travis-ci.org/repo/${USER}%2F${REPO}/requests \
  | tee /tmp/travis-request-output.$$.txt

if grep -q '"@type": "error"' /tmp/travis-request-output.$$.txt; then
   cat /tmp/travis-request-output.$$.txt
   exit 1
elif grep -q 'access denied' /tmp/travis-request-output.$$.txt; then
   cat /tmp/travis-request-output.$$.txt
   exit 1
fi