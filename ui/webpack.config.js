const path = require("path");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const { env } = require("process");

module.exports = {
    mode: "development",
    entry: {
        index: "./src/app.ts",
    },
    devtool: "inline-source-map",
    devServer: {
        allowedHosts: "all",
        host: "0.0.0.0",
        port: "80",
        client: {
            webSocketURL: {
                hostname: "0.0.0.0",
                port: env.WEB_HTTP_PORT
            }
        },
        devMiddleware: {
            publicPath: "/"
        },        
        static: {
            directory: "./dist"
        },
        hot: true
    },
    output: {
        filename: "[name].bundle.js",
        path: path.resolve(__dirname, "dist"),
        clean: true,
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
            {
                test: /\.css$/,
                use: ["style-loader", "css-loader"]
            }
        ]
    },
    plugins: [
        new HtmlWebpackPlugin({
            hash: true,
            template: path.resolve(__dirname, "src", "index.html"),
            filename: path.resolve(__dirname, "dist", "index.html")
        }),
    ],
};
