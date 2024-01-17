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
             checksum: "0ad1d771f095f600d92d26fb425d654d6fb3175ef0b70fb1e68dfc4054ee4d39"
             )
     ]
 )
