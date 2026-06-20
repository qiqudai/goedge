namespace Cnn.Common.Localization;

public interface IMessageLocalizer
{
    string DefaultLanguage { get; }

    string Translate(string key, string? language);
}
