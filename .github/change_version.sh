#! /bin/bash

SED() { [[ "$OSTYPE" == "darwin"* ]] && sed -i '' "$@" || sed -i "$@"; }

echo "previous version was $(git describe --tags $(git rev-list --tags --max-count=1))"
echo "WARNING: This operation will creates version tag and push to github"
read -p "Version? (provide the next x.y.z semver) : " TAG
echo $TAG 
[[ "$TAG" =~ ^[0-9]{1,2}\.[0-9]{1,2}\.[0-9]{1,2}(\.dev)?$ ]] || { echo "Incorrect tag. e.g., 1.2.3 or 1.2.3.dev"; exit 1; } 
IFS="." read -r -a VERSION_ARRAY <<< "$TAG" 
VERSION_STR="${VERSION_ARRAY[0]}.${VERSION_ARRAY[1]}.${VERSION_ARRAY[2]}" 
BUILD_NUMBER=$(( ${VERSION_ARRAY[0]} * 10000 + ${VERSION_ARRAY[1]} * 100 + ${VERSION_ARRAY[2]} )) 
echo "version: ${VERSION_STR}+${BUILD_NUMBER}" 
SED -e "s|<key>CFBundleVersion</key>\s*<string>[^<]*</string>|<key>CFBundleVersion</key><string>${VERSION_STR}</string>|" Info.plist 
SED -e "s|<key>CFBundleShortVersionString</key>\s*<string>[^<]*</string>|<key>CFBundleShortVersionString</key><string>${VERSION_STR}</string>|" Info.plist 
SED "s|ENV VERSION=.*|ENV VERSION=v${TAG}|g" docker/Dockerfile 
git add Info.plist docker/Dockerfile 
git commit -m "release: version ${TAG}" 
echo "creating git tag : v${TAG}" 
git push 
git tag v${TAG} 
git push -u origin HEAD --tags 
echo "Github Actions will detect the new tag and release the new version."