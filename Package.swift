// swift-tools-version: 5.4
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription
import Foundation

let version = "draft"
let baseURL = "https://github.com/hiddify/hiddify-next-core/releases/download/"
let packageURL = baseURL + version + "/hiddify-libcore-ios.xcframework.zip"

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
             checksum: "2e893644b06b9705b05502cf26d7495645944a9d9308c150c2c2940f4a3ca20b"
             )
     ]
 )
