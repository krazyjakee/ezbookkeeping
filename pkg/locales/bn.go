package locales

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
)

var bn = &LocaleTextItems{
	GlobalTextItems: &GlobalTextItems{
		AppName: "ezrbookkeeping",
	},
	DefaultTypes: &DefaultTypes{
		DecimalSeparator:    core.DECIMAL_SEPARATOR_DOT,
		DigitGroupingSymbol: core.DIGIT_GROUPING_SYMBOL_COMMA,
	},
	DataConverterTextItems: &DataConverterTextItems{
		Alipay:       "Alipay",
		WeChatWallet: "ওয়ালেট",
	},
	VerifyEmailTextItems: &VerifyEmailTextItems{
		Title:                     "ইমেল যাচাই করুন",
		SalutationFormat:          "প্রিয় %s,",
		DescriptionAboveBtn:       "আপনার ইমেল ঠিকানা যাচাই করতে অনুগ্রহ করে নিচের লিঙ্কে ক্লিক করুন।",
		VerifyEmail:               "ইমেল যাচাই করুন",
		DescriptionBelowBtnFormat: "আপনি যদি %s অ্যাকাউন্টের জন্য নিবন্ধন না করে থাকেন, তাহলে অনুগ্রহ করে এই ইমেলটি উপেক্ষা করুন। আপনি যদি উপরের লিঙ্কে ক্লিক করতে না পারেন, তাহলে উপরের URL কপি করে আপনার ব্রাউজারে পেস্ট করুন। ইমেল যাচাইকরণ লিঙ্কটি %v মিনিট পরে মেয়াদোত্তীর্ণ হবে।",
	},
	ForgetPasswordMailTextItems: &ForgetPasswordMailTextItems{
		Title:                     "আপনার পাসওয়ার্ড রিসেট করুন",
		SalutationFormat:          "প্রিয় %s,",
		DescriptionAboveBtn:       "আমরা সম্প্রতি আপনার পাসওয়ার্ড রিসেট করার একটি অনুরোধ পেয়েছি। আপনার পাসওয়ার্ড রিসেট করতে নিচের লিঙ্কে ক্লিক করুন।",
		ResetPassword:             "পাসওয়ার্ড রিসেট করুন",
		DescriptionBelowBtnFormat: "আপনি যদি পাসওয়ার্ড রিসেটের অনুরোধ না করে থাকেন, তাহলে অনুগ্রহ করে এই ইমেলটি উপেক্ষা করুন। আপনি যদি উপরের লিঙ্কে ক্লিক করতে না পারেন, তাহলে উপরের URL কপি করে আপনার ব্রাউজারে পেস্ট করুন। পাসওয়ার্ড রিসেট লিঙ্কটি %v মিনিট পরে মেয়াদোত্তীর্ণ হবে।",
	},
}
