module.exports = {
    cache: true,
    entry: "./static/js/app.jsx",
    output: {
        path: __dirname + "/build",
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
