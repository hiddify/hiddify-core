// swift-tools-version: 5.4
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "Libcore",
    platforms: [
        .iOS(.v13) // Minimum platform version
    ],
    products: [
        .library(
            name: "Libcore",
            targets: ["Libcore"]),
    ],
    dependencies: [
        // No dependencies
    ],
    targets: [
        .binaryTarget(
            name: "Libcore",
            url: "https://github.com/hiddify/hiddify-next-core/releases/download/draft/hiddify-libcore-ios.xcframework.zip",
            checksum: "70f84a51508898a706e72ab9eda4af8ab72c321bf79284b38313764b8f2091b2"
        )
    ]
)
