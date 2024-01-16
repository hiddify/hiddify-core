// swift-tools-version: 5.4
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

func fetchChecksum(from url: String) throws -> String {
    guard let checksumURL = URL(string: url) else {
        throw NSError(domain: "Invalid URL", code: 0, userInfo: nil)
    }

    let checksumString = try String(contentsOf: checksumURL)
    return checksumString.trimmingCharacters(in: .whitespacesAndNewlines)
}

let version = "draft"
let baseURL = "https://github.com/hiddify/hiddify-next-core/releases/download/"
let packageURL = "\(baseURL)\(version)/hiddify-libcore-ios.xcframework.zip"
let checksumURL = "\(packageURL).sha256"

do {
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
                url: packageURL,
                checksum: try fetchChecksum(from: checksumURL)
            )
        ]
    )
} catch {
    // Handle URL or checksum fetch errors
    print("Error: \(error)")
    // You might want to exit or handle the error in a way suitable for your application
    // exit(1)
}
