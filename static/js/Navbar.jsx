/** @jsx React.DOM */

'use strict';

var React = require('react'),
    RRouter = require('rrouter'),
    Link = RRouter.Link;

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

module.exports = Navbar;
