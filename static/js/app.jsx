/** @jsx React.DOM */

'use strict';

var React = require('react'),
    Fluxxor = require('fluxxor'),
    RRouter = require('rrouter'),
    Routes = RRouter.Routes,
    Route = RRouter.Route,
    Link = RRouter.Link;


var ItemsStore = Fluxxor.createStore({
    actions: {
        'ADD_ITEM': 'onAddItem',
    },

    initialize: function() {
        this.items = [];
    },

    onAddItem: function(payload) {
        this.items.push({
            url:      payload.url,
            selector: payload.selector,
            schedule: payload.schedule,
        });
        this.emit('change');
    },

    getState: function() {
        return {
            items: this.items,
        };
    },
});


var LogsStore = Fluxxor.createStore({
    actions: {
        'ADD_LOG': 'onAddLog',
        'CLEAR_LOGS': 'onClearLogs',
    },

    initialize: function() {
        this.logs = [];
    },

    onAddLog: function(payload) {
        this.logs.push({
            level:   payload.level,
            message: payload.message,
        });
        this.emit('change');
    },

    onClearLogs: function() {
        this.logs = [];
        this.emit('change');
    },

    getState: function() {
        return {
            logs: this.logs,
        };
    },
});

var actions = {
    addItem: function(url, selector, schedule) {
        this.dispatch("ADD_ITEM", {
            url: url,
            selector: selector,
            schedule: schedule,
        });
    },

    addLog: function(level, message) {
        this.dispatch("ADD_LOG", {
            level:   level,
            message: message,
        });
    },
    clearLogs: function() {
        this.dispatch("CLEAR_LOGS");
    },
};


var stores = {
    ItemStore: new ItemsStore(),
    LogStore: new LogsStore(),
};


var flux = new Fluxxor.Flux(stores, actions);
window.flux = flux;


var FluxMixin = Fluxxor.FluxMixin(React),
    FluxChildMixin = Fluxxor.FluxChildMixin(React),
    StoreWatchMixin = Fluxxor.StoreWatchMixin;


var ActionButton = React.createClass({
    propTypes: {
        type:   React.PropTypes.string.isRequired,
        icon:   React.PropTypes.string.isRequired,
        action: React.PropTypes.func.isRequired,
    },

    render: function() {
        var buttonClasses = 'btn btn-xs ' + this.props.type,
            iconClasses   = 'glyphicon ' + this.props.icon;

        return (
            <button className={buttonClasses} onClick={this.props.action}>
                <span className={iconClasses}></span>
            </button>
        );
    },
});


var ItemRow = React.createClass({
    propTypes: {
        item: React.PropTypes.shape({
            url:      React.PropTypes.string.isRequired,
            selector: React.PropTypes.string.isRequired,
            schedule: React.PropTypes.string.isRequired,
        }),
    },

    render: function() {
        return (
            <tr>
                <td>Label</td>
                <td>{this.props.item.url}</td>
                <td>{this.props.item.selector}</td>
                <td>{this.props.item.schedule}</td>
                <td>Last Successful Check</td>
                <td>Hash</td>
                <td>
                    <ActionButton
                        type="btn-success"
                        icon="glyphicon-ok"
                        action={this.onMarkRead} />
                    <ActionButton
                        type="btn-danger"
                        icon="glyphicon-remove"
                        action={this.onDeleteItem} />
                    <ActionButton
                        type="btn-primary"
                        icon="glyphicon-refresh"
                        action={this.onRefreshItem} />
                </td>
            </tr>
        );
    },

    onMarkRead: function() {
        console.log("Marking as read");
    },

    onDeleteItem: function() {
        console.log("Deleting item");
    },

    onRefreshItem: function() {
        console.log("Refreshing item");
    },
});


var NewItemForm = React.createClass({
    mixins: [FluxChildMixin],

    getInitialState: function() {
        return {
            url:      '',
            selector: '',
            schedule: '',
        };
    },

    handleURLChange: function(event) {
        this.setState({url: event.target.value});
    },

    handleSelectorChange: function(event) {
        this.setState({selector: event.target.value});
    },

    handleScheduleChange: function(event) {
        this.setState({schedule: event.target.value});
    },

    render: function() {
        return (
            <form className="form-inline">
                <input
                    type="text"
                    className="form-control"
                    placeholder="URL to check"
                    value={this.state.url}
                    onChange={this.handleURLChange}
                    />
                <input
                    type="text"
                    className="form-control"
                    placeholder="Selector to monitor"
                    value={this.state.selector}
                    onChange={this.handleSelectorChange}
                    />
                <input
                    type="text"
                    className="form-control"
                    placeholder="Schedule"
                    value={this.state.schedule}
                    onChange={this.handleScheduleChange}
                    />

                <button className="btn btn-primary" onClick={this.onAdd}>Add</button>
                <button className="btn btn-default" onClick={this.onClear}>Clear</button>
            </form>
        );
    },

    onAdd: function() {
        this.getFlux().actions.addItem(this.state.url,
                                       this.state.selector,
                                       this.state.schedule);
    },

    onClear: function() {
        this.setState({
            url:      '',
            selector: '',
            schedule: '',
        });
    },
});


var Main = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("ItemStore")],

    getStateFromFlux: function() {
        return this.getFlux().store("ItemStore").getState();
    },

    render: function() {
        return (
            <div>
                <h2>Main</h2>

                <table className="table">
                    <thead>
                        <tr>
                            <th>Status</th>
                            <th>URL</th>
                            <th>Selector</th>
                            <th>Schedule</th>
                            <th>Last Successful Check</th>
                            <th>Hash</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                    {this.state.items.map(function(item, i) {
                        return <ItemRow key={i} item={item} />
                    })}
                    </tbody>
                </table>

                <NewItemForm />
            </div>
        );
    },
});


var LogRow = React.createClass({
    propTypes: {
        log: React.PropTypes.shape({
            level:   React.PropTypes.string.isRequired,
            message: React.PropTypes.string.isRequired,
        }),
    },

    render: function() {
        return (
            <tr>
                <td>Time</td>
                <td>{this.props.log.level}</td>
                <td>{this.props.log.message}</td>
                <td>Fields</td>
            </tr>
        );
    },
});


var Logs = React.createClass({
    mixins: [FluxMixin, StoreWatchMixin("LogStore")],

    getStateFromFlux: function() {
        return this.getFlux().store("LogStore").getState();
    },

    render: function() {
        return (
            <div>
                <h2>Logs</h2>
                <button onClick={this.onClick} className="btn btn-primary">Clear Logs</button>

                <table className="table">
                    <thead>
                        <tr>
                            <th>Time</th>
                            <th>Level</th>
                            <th>Message</th>
                            <th>Fields</th>
                        </tr>
                    </thead>

                    <tbody>
                    {this.state.logs.map(function(log, i) {
                        return <LogRow key={i} log={log} />
                    })}
                    </tbody>
                </table>
            </div>
        );
    },

    onClick: function() {
        this.getFlux().actions.clearLogs();
    },
});


var Navbar = React.createClass({
    render: function() {
        return (
            <div className="navbar navbar-default navbar-static-top" role="navigation">
                <div className="container">
                    <div className="navbar-header">
                        <button type="button" className="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
                            <span className="sr-only">Toggle navigation</span>
                            <span className="icon-bar"></span>
                            <span className="icon-bar"></span>
                            <span className="icon-bar"></span>
                        </button>
                        <a className="navbar-brand" href="/">Site Monitor</a>
                    </div>
                    <div className="navbar-collapse collapse">
                        <ul className="nav navbar-nav">
                            <li><Link to="/main">Home</Link></li>
                            <li><a href="/stats">Statistics</a></li>
                            <li>
                                <Link to="/logs">
                                    <span className="badge pull-right">{this.props.numLogs}</span>
                                    Logs&nbsp;
                                </Link>
                            </li>
                            <li><a href="/about">About</a></li>
                        </ul>
                    </div>
                </div>
            </div>
        );
    },
});


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
