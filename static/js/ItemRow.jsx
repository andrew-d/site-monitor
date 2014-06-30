/** @jsx React.DOM */

'use strict';

var React = require('react'),
    Fluxxor = require('fluxxor'),
    moment = require('moment');

var FluxChildMixin = Fluxxor.FluxChildMixin(React);

var ActionButton = require('./ActionButton.jsx');

var ItemRow = React.createClass({
    mixins: [FluxChildMixin],

    propTypes: {
        item: React.PropTypes.shape({
            id:           React.PropTypes.number.isRequired,
            url:          React.PropTypes.string.isRequired,
            selector:     React.PropTypes.string.isRequired,
            schedule:     React.PropTypes.string.isRequired,
            last_checked: React.PropTypes.string.isRequired,
            last_hash:    React.PropTypes.string.isRequired,
            seen:         React.PropTypes.bool.isRequired,
        }),
    },

    render: function() {
        var label;
        if( this.props.item.seen ) {
            label = <span className="label label-default">Seen</span>;
        } else {
            label = <span className="label label-primary">Changed</span>;
        }

        var hash;
        if( this.props.item.last_hash ) {
            hash = (
                <span title={this.props.item.last_hash}>
                    {this.props.item.last_hash.slice(0, 8)}
                </span>
            );
        } else {
            hash = <span>(none)</span>;
        }

        var last_checked;
        var m = moment(this.props.item.last_checked, "YYYY-MM-DDTHH:mm:ssZ");

        // Note: can't check the value of "last_checked", since Go will happily
        // serialize the zero time for us.  We check the hash instead.
        // TODO: look into fixing on Go's end.
        if( !this.props.item.last_hash ) {
            last_checked = <span>(never)</span>;
        } else if( !m.isValid() ) {
            last_checked = (
                <span title="(invalid format)">
                    {this.props.item.last_checked}
                </span>
            );
        } else {
            last_checked = (
                <span title={this.props.item.last_checked}>
                    {m.fromNow()}
                </span>
            );
        }

        return (
            <tr>
                <td>{label}</td>
                <td>{this.props.item.url}</td>
                <td>{this.props.item.selector}</td>
                <td>{this.props.item.schedule}</td>
                <td>{last_checked}</td>
                <td>
                    {hash}
                </td>
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
        this.getFlux().actions.markItemRead(this.props.item.id);
    },

    onDeleteItem: function() {
        this.getFlux().actions.deleteItem(this.props.item.id);
    },

    onRefreshItem: function() {
        this.getFlux().actions.refreshItem(this.props.item.id);
    },
});

module.exports = ItemRow;
