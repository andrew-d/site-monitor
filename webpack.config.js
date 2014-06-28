module.exports = {
    cache: true,
    entry: "./static/js/App.jsx",
    output: {
        path: __dirname + "/build/js",
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
