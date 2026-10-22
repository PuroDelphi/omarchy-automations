// Test-only QML extension. Offscreen clipboard; never touches the user's clipboard.
#include <QClipboard>
#include <QGuiApplication>
#include <QMimeData>
#include <QQmlExtensionPlugin>
#include <QTimer>
#include <QtQml>

static int reads = 0;
class OfferedText : public QMimeData {
public:
    QStringList formats() const override { return {QStringLiteral("text/plain")}; }
protected:
    QVariant retrieveData(const QString &, QMetaType) const override {
        ++reads;
        return QStringLiteral("bounded clipboard regression fixture");
    }
};

class ClipboardProbe : public QObject {
    Q_OBJECT
public:
    using QObject::QObject;
    Q_INVOKABLE int readCount() const { return reads; }
    Q_INVOKABLE void offer() {
        QGuiApplication::clipboard()->setMimeData(new OfferedText);
    }
    Q_INVOKABLE void exhaustMemory() {
        // Called only inside the verified 512 MiB editor cgroup, never the host.
        QTimer::singleShot(100, this, [] {
            QList<QByteArray> retained;
            for (int i = 0; i < 128; ++i)
                retained.append(QByteArray(8 * 1024 * 1024, 'x'));
        });
    }
};

class ClipboardProbePlugin : public QQmlExtensionPlugin {
    Q_OBJECT
    Q_PLUGIN_METADATA(IID "org.qt-project.Qt.QQmlExtensionInterface/1.0")
public:
    void registerTypes(const char *uri) override {
        qmlRegisterType<ClipboardProbe>(uri, 1, 0, "ClipboardProbe");
    }
};
#include "clipboard_probe.moc"
