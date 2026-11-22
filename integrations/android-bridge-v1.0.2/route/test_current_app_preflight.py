import tempfile
from pathlib import Path
import unittest

from current_app_preflight import TARGETS, scan


class AuditTests(unittest.TestCase):
    def test_readonly_and_redaction(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            kt = root / 'android/app/src/main/kotlin/com/example'
            kt.mkdir(parents=True)
            (root / 'lib').mkdir()
            for name in TARGETS:
                (kt / name).write_text('class ' + name[:-3] + ' { val password = "DO_NOT_PRINT" }\n')
            dart = root / 'lib' / 'main.dart'
            dart.write_text('BridgeVpnSession // secret abc123\n')
            elf = root / 'android/app/src/main/jniLibs/arm64-v8a/libtervonbridge.so'
            elf.parent.mkdir(parents=True)
            elf.write_bytes(b'fake-core')
            before = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob('*') if p.is_file()}
            result = scan(root)
            after = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob('*') if p.is_file()}
            self.assertEqual(before, after)
            self.assertTrue(all(len(result['seven_targets'][name]) == 1 for name in TARGETS))
            self.assertFalse(result['arm64_core']['matches_confirmed_diagnostic_core'])
            self.assertEqual(2, len(result['callsite_locations_only']['BridgeVpnSession']))
            self.assertNotIn('DO_NOT_PRINT', str(result))
            self.assertNotIn('abc123', str(result))

    def test_missing_android_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError, 'NOT_ANDROID_PROJECT_ROOT'):
                scan(Path(directory))

    def test_source_symlink_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            kt = root / 'android/app/src/main/kotlin'
            kt.mkdir(parents=True)
            actual = root / 'real.kt'
            actual.write_text('class Demo')
            (kt / 'BridgeVpnSession.kt').symlink_to(actual)
            with self.assertRaisesRegex(ValueError, 'UNSAFE_SOURCE_ENTRY'):
                scan(root)


if __name__ == '__main__':
    unittest.main()
