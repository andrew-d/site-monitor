/** @jsx React.DOM */

'use strict';

var React = require('react');

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

module.exports = ActionButton;
