module.exports = {
    cache: true,
    entry: "./js/app.jsx",
    output: {
        path: __dirname + "/js",
        filename: "bundle.js"
    },
    devtool: "source-map",
    module: {
        loaders: [
            //{ test: /\.less$/, loader: "style!css!less" },
            { test: /\.jsx$/, loader: "jsx-loader" },
            { test: /\.json$/, loader: "json" }
        ]
    }
};
