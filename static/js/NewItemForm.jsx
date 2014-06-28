/** @jsx React.DOM */

var React = require('react'),
    Fluxxor = require('fluxxor');

var FluxChildMixin = Fluxxor.FluxChildMixin(React);

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

    onAdd: function(e) {
        e.preventDefault();
        e.stopPropagation();
        this.getFlux().actions.addItem(this.state.url,
                                       this.state.selector,
                                       this.state.schedule);
    },

    onClear: function(e) {
        e.preventDefault();
        e.stopPropagation();
        this.setState({
            url:      '',
            selector: '',
            schedule: '',
        });
    },
});

module.exports = NewItemForm;
