// @flow

var Track = React.createClass({
	mixins: [Reflux.listenTo(Stores.tracks, 'update')],
	play: function() {
		if (this.props.isqueue) {
			POST('/api/cmd/play_idx?idx=' + this.props.idx);
		} else {
			var params = mkcmd([
				'clear',
				'add-' + this.props.id.UID
			]);
			POST('/api/queue/change', params, function() {
				POST('/api/cmd/play');
			});
		}
	},
	getInitialState: function() {
		if (this.props.info) {
			return {
				info: this.props.info
			};
		}
		var d = Lookup(this.props.id);
		if (d) {
			return {
				info: d.Info
			};
		}
		return {};
	},
	update: function() {
		this.setState(this.getInitialState());
	},
	over: function() {
		this.setState({over: true});
	},
	out: function() {
		this.setState({over: false});
	},
	dequeue: function() {
		var params = mkcmd([
			'rem-' + this.props.idx
		]);
		POST('/api/queue/change', params);
	},
	append: function() {
		var params = mkcmd([
			'add-' + this.props.id.UID
		]);
		POST('/api/queue/change', params);
	},
	render: function() {
		var info = this.state.info;
		if (!info) {
			return (
				<tr>
					<td>{this.props.id}</td>
				</tr>
			);
		}
		var control;
		var track;
		if (this.state.over) {
			if (this.props.isqueue) {
				control = (
					<div>
						<button onClick={this.dequeue}>x</button>
					</div>
				);
			} else {
				control = (
					<div>
						<button onClick={this.append}>+</button>
					</div>
				);
			}
			track = <button onClick={this.play}>&#x25b6;</button>;
		} else {
			track = info.Track || '';
		}
		return (
			<tr onMouseEnter={this.over} onMouseLeave={this.out}>
				<td className="control">{track}</td>
				<td>{info.Title}</td>
				<td className="control">{control}</td>
				<td><Time time={info.Time} /></td>
				<td><Link to="artist" params={info}>{info.Artist}</Link></td>
				<td><Link to="album" params={info}>{info.Album}</Link></td>
			</tr>
		);
	}
});

var Tracks = React.createClass({
	mkparams: function() {
		return _.map(this.props.tracks, function(t) {
			return 'add-' + t.ID.UID;
		});
	},
	play: function() {
		var params = this.mkparams();
		params.unshift('clear');
		POST('/api/queue/change', mkcmd(params), function() {
			POST('/api/cmd/play');
		});
	},
	add: function() {
		var params = this.mkparams();
		POST('/api/queue/change', mkcmd(params));
	},
	render: function() {
		var tracks = _.map(this.props.tracks, function(t, idx) {
			return <Track key={idx + '-' + t.ID.UID} id={t.ID} info={t.Info} idx={idx} isqueue={this.props.isqueue} />;
		}.bind(this));
		var queue;
		if (!this.props.isqueue) {
			queue = (
				<div>
					<button onClick={this.play}>play</button>
					<button onClick={this.add}>add</button>
				</div>
			);
		};
		return (
			<div>
				{queue}
				<table className="u-full-width tracks">
					<thead>
						<tr>
							<th>#</th>
							<th>Name</th>
							<th></th>
							<th>Time</th>
							<th>Artist</th>
							<th>Album</th>
						</tr>
					</thead>
					<tbody>{tracks}</tbody>
				</table>

			</div>
		);
	}
});

var TrackList = React.createClass({
	mixins: [Reflux.listenTo(Stores.tracks, 'setState')],
	getInitialState: function() {
		return Stores.tracks.data || {};
	},
	render: function() {
		return <Tracks tracks={this.state.Tracks} />;
	}
});

function searchClass(field) {
	return React.createClass({
		mixins: [Reflux.listenTo(Stores.tracks, 'setState')],
		render: function() {
			if (!Stores.tracks.data) {
				return null;
			}
			var tracks = [];
			var prop = this.props.params[field];
			_.each(Stores.tracks.data.Tracks, function(val) {
				if (val.Info[field] == prop) {
					tracks.push(val);
				}
			});
			return <Tracks tracks={tracks} />;
		}
	});
}

var Artist = searchClass('Artist');
var Album = searchClass('Album');
