/** @jsx React.DOM */

'use strict';

var React = require('react'),
    Fluxxor = require('fluxxor'),
    RRouter = require('rrouter'),
    Routes = RRouter.Routes,
    Route = RRouter.Route,
    Link = RRouter.Link;

var FluxMixin = Fluxxor.FluxMixin(React),
    StoreWatchMixin = Fluxxor.StoreWatchMixin;

var flux = require('./stores/stores.js');

var Navbar = require('./Navbar.jsx'),
    Logs   = require('./Logs.jsx'),
    Main   = require('./Main.jsx');

var App = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("LogStore")],

    getStateFromFlux: function() {
        return {
            numLogs: this.getFlux().store("LogStore").getState().logs.length,
        };
    },

    render: function() {
        return (
            <div>
                <Navbar numLogs={this.state.numLogs} />
                <div className="container">
                    {this.props.children}
                </div>
            </div>
        )
    },
});


var routes = (
    <Routes>
        <Route name="main" path="/" view={Main} flux={flux} />
        <Route name="logs" path="/logs" view={Logs} flux={flux} />
    </Routes>
);


RRouter.HashRouting.start(routes, function(view) {
    React.renderComponent(<App flux={flux}>{view}</App>, document.body)
})
