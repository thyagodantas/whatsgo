// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsgo

import (
	"context"

	waBinary "github.com/thyagodantas/whatsgo/binary"
	"github.com/thyagodantas/whatsgo/types"
	"github.com/thyagodantas/whatsgo/types/events"
)

// Calling support in whatsgo is limited to the receive side: the library
// parses <call> stanzas the server sends and dispatches them as events
// (CallOffer, CallAccept, CallTerminate, etc.) and exposes RejectCall so
// the bot can refuse inbound calls. Enough for monitoring, logging, and
// rejecting.
//
// Full call lifecycle (answer, place outbound calls, audio, video) lives
// in the external github.com/purpshell/meowcaller library. See CALL.md
// in the project root for the integration pattern.

func (cli *Client) handleCallEvent(ctx context.Context, node *waBinary.Node) {
	defer cli.maybeDeferredAck(ctx, node)()

	if len(node.GetChildren()) != 1 {
		cli.dispatchEvent(&events.UnknownCallEvent{Node: node})
		return
	}
	ag := node.AttrGetter()
	child := node.GetChildren()[0]
	cag := child.AttrGetter()
	basicMeta := types.BasicCallMeta{
		From:        ag.JID("from"),
		Timestamp:   ag.UnixTime("t"),
		CallCreator: cag.JID("call-creator"),
		CallID:      cag.String("call-id"),
		GroupJID:    cag.OptionalJIDOrEmpty("group-jid"),
	}
	if basicMeta.CallCreator.Server == types.HiddenUserServer {
		basicMeta.CallCreatorAlt = cag.OptionalJIDOrEmpty("caller_pn")
	} else {
		basicMeta.CallCreatorAlt = cag.OptionalJIDOrEmpty("caller_lid")
	}
	switch child.Tag {
	case "offer":
		cli.dispatchEvent(&events.CallOffer{
			BasicCallMeta: basicMeta,
			CallRemoteMeta: types.CallRemoteMeta{
				RemotePlatform: ag.String("platform"),
				RemoteVersion:  ag.String("version"),
			},
			Data: &child,
		})
	case "offer_notice":
		cli.dispatchEvent(&events.CallOfferNotice{
			BasicCallMeta: basicMeta,
			Media:         cag.String("media"),
			Type:          cag.String("type"),
			Data:          &child,
		})
	case "relaylatency":
		cli.dispatchEvent(&events.CallRelayLatency{
			BasicCallMeta: basicMeta,
			Data:          &child,
		})
	case "accept":
		cli.dispatchEvent(&events.CallAccept{
			BasicCallMeta: basicMeta,
			CallRemoteMeta: types.CallRemoteMeta{
				RemotePlatform: ag.String("platform"),
				RemoteVersion:  ag.String("version"),
			},
			Data: &child,
		})
	case "preaccept":
		cli.dispatchEvent(&events.CallPreAccept{
			BasicCallMeta: basicMeta,
			CallRemoteMeta: types.CallRemoteMeta{
				RemotePlatform: ag.String("platform"),
				RemoteVersion:  ag.String("version"),
			},
			Data: &child,
		})
	case "transport":
		cli.dispatchEvent(&events.CallTransport{
			BasicCallMeta: basicMeta,
			CallRemoteMeta: types.CallRemoteMeta{
				RemotePlatform: ag.String("platform"),
				RemoteVersion:  ag.String("version"),
			},
			Data: &child,
		})
	case "terminate":
		cli.dispatchEvent(&events.CallTerminate{
			BasicCallMeta: basicMeta,
			Reason:        cag.String("reason"),
			Data:          &child,
		})
	case "reject":
		cli.dispatchEvent(&events.CallReject{
			BasicCallMeta: basicMeta,
			Data:          &child,
		})
	default:
		cli.dispatchEvent(&events.UnknownCallEvent{Node: node})
	}
}

// RejectCall rejects an incoming call.
func (cli *Client) RejectCall(ctx context.Context, callFrom types.JID, callID string) error {
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return ErrNotLoggedIn
	}
	ownID, callFrom = ownID.ToNonAD(), callFrom.ToNonAD()
	rejectNode := waBinary.Node{
		Tag:     "reject",
		Attrs:   waBinary.Attrs{"call-id": callID, "call-creator": callFrom, "count": "0"},
		Content: nil,
	}
	if token, err := cli.ensureTCToken(ctx, callFrom); err != nil {
		cli.Log.Warnf("Failed to get privacy token for call reject to %s: %v", callFrom, err)
	} else if len(token) > 0 {
		rejectNode.Content = []waBinary.Node{{
			Tag:     "tctoken",
			Content: token,
		}}
	}
	return cli.sendNode(ctx, waBinary.Node{
		Tag:     "call",
		Attrs:   waBinary.Attrs{"id": cli.GenerateMessageID(), "from": ownID, "to": callFrom},
		Content: []waBinary.Node{rejectNode},
	})
}