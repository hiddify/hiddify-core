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
             checksum: "e42d176d32198bc33ae4f8aafa24814934bbc93cbc8747aa8a0353e8341dc3ff"
             )
     ]
 )
