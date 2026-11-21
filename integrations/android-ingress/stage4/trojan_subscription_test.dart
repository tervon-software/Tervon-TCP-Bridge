import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:tervonvpndpi/subscriptions/subscription_importer.dart';
import 'package:tervonvpndpi/vpn/sub_parser.dart';

void main() {
  const trojanTcp = 'trojan://example-password@node.example:41877?security=tls&type=tcp#Trojan';
  const trojanWs = 'trojan://password@node.example:42011?security=tls&type=ws&host=front.example#WS';
  const trojanGrpc = 'trojan://password@node.example:42199?security=tls&type=grpc#GRPC';
  const trojanXhttp = 'trojan://password@node.example:41999?security=tls&type=xhttp#XHTTP';

  test('Trojan subscription imports and selects all supported TCP transports', () {
    final raw = [trojanTcp, trojanWs, trojanGrpc, trojanXhttp].join('\n');
    final imported = SubscriptionImporter.import(raw);
    expect(imported.links.length, 4);
    final types = SubParser.getAvailableProtocolTypes(raw);
    expect(types, containsAll(['trojan_tcp', 'trojan_ws', 'trojan_grpc', 'trojan_xhttp']));
    expect(SubParser.displayName('trojan_tcp'), 'Trojan TCP/TLS');
    final encoded = base64Encode(utf8.encode(raw));
    expect(SubParser.getAvailableProtocolTypes(encoded), types);
  });

  test('Unsupported Trojan UDP/H3 or disabled TLS is not selectable', () {
    expect(SubParser.getAvailableProtocolTypes(
      'trojan://p@vpn.example:40211?type=quic'), isEmpty);
    expect(SubParser.getAvailableProtocolTypes(
      'trojan://p@vpn.example:40211?security=none&type=tcp'), isEmpty);
  });
}
